package admins

import (
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

func Create(c *gin.Context) {
	user := c.MustGet(middleware.User).(*session.Session)
	if user == nil {
		errors.SendErrorResponse(c, errors.ErrSessionDidntPass, errors.StatusInternalError)
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

	var isOwner bool
	var isUserAdmin bool
	var isDm bool
	var isUserInGuild bool

	if err := db.Db.QueryRow(`
	SELECT EXISTS (SELECT 1 FROM userguilds WHERE user_id = $1 AND guild_id = $2 AND owner = true), 
	EXISTS (SELECT 1 FROM guilds WHERE id = $2 AND dm = true),
	EXISTS (SELECT 1 FROM userguilds WHERE guild_id = $2 AND user_id = $3 AND admin = true),
	EXISTS (SELECT 1 FROM userguilds WHERE guild_id = $2 AND user_id = $3)
	`, user.Id, guildId, userId).Scan(&isOwner, &isDm, &isUserAdmin, &isUserInGuild); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	if isDm {
		errors.SendErrorResponse(c, errors.ErrGuildIsDm, errors.StatusGuildIsDm)
		return
	}
	if !isOwner {
		errors.SendErrorResponse(c, errors.ErrNotAuthorised, errors.StatusNotAuthorised)
		return
	}
	if isUserAdmin {
		errors.SendErrorResponse(c, errors.ErrUserAlreadyAdmin, errors.StatusUserAlreadyAdmin)
		return
	}
	if !isUserInGuild {
		errors.SendErrorResponse(c, errors.ErrNotInGuild, errors.StatusNotInGuild)
		return
	}

	if _, err := db.Db.Exec("UPDATE userguilds SET admin = true WHERE user_id = $1 AND guild_id = $2", userId, guildId); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	intUserId, err := strconv.ParseInt(userId, 10, 64)
	if err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	intGuildId, err := strconv.ParseInt(guildId, 10, 64)
	if err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	rows, err := db.Db.Query("SELECT user_id FROM userguilds WHERE guild_id = $1 AND admin = true OR owner = true", guildId)
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
			Event: events.MEMBER_ADMIN_ADD,
		}
		wsclient.Pools.BroadcastClient(adminUserId, res)
	}
	c.Status(http.StatusNoContent)
}
