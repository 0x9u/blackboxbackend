package bans

import (
	"net/http"
	"regexp"
	"strconv"

	"github.com/asianchinaboi/backendserver/internal/api/middleware"
	"github.com/asianchinaboi/backendserver/internal/db"
	"github.com/asianchinaboi/backendserver/internal/errors"
	"github.com/asianchinaboi/backendserver/internal/events"
	"github.com/asianchinaboi/backendserver/internal/logger"
	"github.com/asianchinaboi/backendserver/internal/session"
	"github.com/asianchinaboi/backendserver/internal/wsclient"
	"github.com/gin-gonic/gin"
)

func Unban(c *gin.Context) {
	user := c.MustGet(middleware.User).(*session.Session)
	if user == nil {
		errors.SendErrorResponse(c, errors.ErrSessionDidntPass, errors.StatusInternalError)
		return
	}
	if !user.Perms.Admin && !user.Perms.Guilds.Edit {
		errors.SendErrorResponse(c, errors.ErrNotAuthorised, errors.StatusNotAuthorised)
		return
	}

	guildId := c.Param("guildId")
	if match, err := regexp.MatchString("^[0-9]+$", guildId); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	} else if !match {
		errors.SendErrorResponse(c, errors.ErrRouteParamInvalid, errors.StatusRouteParamInvalid)
		return
	}

	userId := c.Param("userId")
	if match, err := regexp.MatchString("^[0-9]+$", userId); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	} else if !match {
		errors.SendErrorResponse(c, errors.ErrRouteParamInvalid, errors.StatusRouteParamInvalid)
		return
	}

	var isBanned bool

	if err := db.Db.QueryRow("SELECT EXISTS (SELECT 1 FROM userguilds WHERE guild_id=$1 AND user_id=$2 AND banned = true)", guildId, userId).Scan(&isBanned); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	if !isBanned {
		errors.SendErrorResponse(c, errors.ErrUserNotBanned, errors.StatusUserNotBanned)
		return
	}

	if _, err := db.Db.Exec("DELETE FROM userguilds WHERE guild_id=$1 AND user_id=$2", guildId, userId); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}
	intGuildId, err := strconv.ParseInt(guildId, 10, 64)
	if err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}
	intUserId, err := strconv.ParseInt(userId, 10, 64)
	if err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	rows, err := db.Db.Query("SELECT user_id FROM userguilds WHERE guild_id = $1 AND (owner = true OR admin = true)", guildId)
	if err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var adminUserId int64
		rows.Scan(&adminUserId)
		res := wsclient.DataFrame{
			Op: wsclient.TYPE_DISPATCH,
			Data: events.Member{
				GuildId: intGuildId,
				UserInfo: events.User{
					UserId: intUserId,
				},
			},
			Event: events.MEMBER_BAN_REMOVE,
		}
		wsclient.Pools.BroadcastClient(adminUserId, res)
	}
	c.Status(http.StatusNoContent)
}
