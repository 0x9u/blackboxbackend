package bans

import (
	"database/sql"
	"net/http"
	"regexp"
	"strconv"

	"github.com/asianchinaboi/backendserver/internal/api/middleware"
	"github.com/asianchinaboi/backendserver/internal/db"
	"github.com/asianchinaboi/backendserver/internal/errors"
	"github.com/asianchinaboi/backendserver/internal/events"
	"github.com/asianchinaboi/backendserver/internal/session"
	"github.com/asianchinaboi/backendserver/internal/wsclient"
	"github.com/gin-gonic/gin"
)

func Ban(c *gin.Context) {
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

	var isBanned bool
	var userExists bool

	if err := db.Db.QueryRow("SELECT EXISTS (SELECT 1 FROM userguilds WHERE guild_id = $1 AND user_id=$2 AND banned = true), EXISTS (SELECT 1 FROM users WHERE id = $2)", guildId, userId).Scan(&isBanned, &userExists); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}
	if isBanned {
		errors.SendErrorResponse(c, errors.ErrUserNotBanned, errors.StatusUserNotBanned)
		return
	}
	if !userExists {
		errors.SendErrorResponse(c, errors.ErrUserNotFound, errors.StatusUserNotFound)
		return
	}

	if _, err := db.Db.Exec("UPDATE userguilds SET banned=true WHERE guild_id=$1 AND user_id=$2", guildId, userId); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}
	var username string
	var imageId sql.NullInt64
	if err := db.Db.QueryRow("SELECT username, files.id FROM users LEFT JOIN files ON files.user_id = users.id WHERE users.id=$1", userId).Scan(&username, &imageId); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	var resImageId int64
	if imageId.Valid {
		resImageId = imageId.Int64
	} else {
		resImageId = -1
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
					UserId:  intUserId,
					Name:    username,
					ImageId: resImageId,
				},
			},
			Event: events.MEMBER_BAN_ADD,
		}
		wsclient.Pools.BroadcastClient(adminUserId, res)
	}

	banRes := wsclient.DataFrame{
		Op: wsclient.TYPE_DISPATCH,
		Data: events.Guild{
			GuildId: intGuildId,
		},
		Event: events.GUILD_DELETE,
	}
	guildRes := wsclient.DataFrame{
		Op: wsclient.TYPE_DISPATCH,
		Data: events.Member{
			GuildId: intGuildId,
			UserInfo: events.User{
				UserId: intUserId,
			},
		},
		Event: events.MEMBER_REMOVE,
	}
	wsclient.Pools.BroadcastClient(intUserId, banRes)
	wsclient.Pools.RemoveUserFromGuildPool(intGuildId, intUserId)
	wsclient.Pools.BroadcastGuild(intGuildId, guildRes)
	c.Status(http.StatusNoContent)
}
