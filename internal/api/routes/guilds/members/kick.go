package members

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

func Kick(c *gin.Context) {
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

	if intUserId == user.Id {
		errors.SendErrorResponse(c, errors.ErrCantKickBanSelf, errors.StatusCantKickBanSelf)
		return
	}

	var isAdmin bool
	var isOwner bool
	var isUserOwner bool
	var isUserAdmin bool
	var isUserInGuild bool
	var isDm bool

	if err := db.Db.QueryRow(`
	SELECT EXISTS (SELECT 1 FROM userguilds WHERE guild_id=$1 AND user_id=$3 AND owner=true), 
	EXISTS (SELECT 1 FROM guilds WHERE id = $1 AND dm = true),
	EXISTS (SELECT 1 FROM userguilds WHERE guild_id=$1 AND user_id=$2 AND admin=true),
	EXISTS (SELECT 1 FROM userguilds WHERE guild_id=$1 AND user_id=$2 AND owner=true),
	EXISTS (SELECT 1 FROM userguilds WHERE guild_id=$1 AND user_id=$3 AND admin=true),
	EXISTS (SELECT 1 FROM userguilds WHERE guild_id=$1 AND user_id=$3)
	`, guildId, user.Id, userId).Scan(&isUserOwner, &isDm, &isAdmin, &isOwner, &isUserAdmin, &isUserInGuild); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	if isDm {
		errors.SendErrorResponse(c, errors.ErrGuildIsDm, errors.StatusGuildIsDm)
		return
	}

	if !isAdmin && !isOwner {
		errors.SendErrorResponse(c, errors.ErrNotGuildAuthorised, errors.StatusNotGuildAuthorised)
		return
	}

	if !isUserInGuild {
		errors.SendErrorResponse(c, errors.ErrNotInGuild, errors.StatusNotInGuild)
		return
	}

	if isUserAdmin && !isOwner {
		errors.SendErrorResponse(c, errors.ErrNotAuthorised, errors.StatusNotAuthorised)
		return
	}

	if _, err := db.Db.Exec("DELETE FROM userguilds WHERE guild_id=$1 AND user_id=$2", guildId, userId); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	kickRes := wsclient.DataFrame{
		Op: wsclient.TYPE_DISPATCH,
		Data: events.Guild{
			GuildId: intGuildId,
		},
		Event: events.DELETE_GUILD,
	}
	guildRes := wsclient.DataFrame{
		Op: wsclient.TYPE_DISPATCH,
		Data: events.Member{
			GuildId: intGuildId,
			UserInfo: events.User{
				UserId: intUserId,
			},
		},
		Event: events.REMOVE_USER_GUILDLIST,
	}
	wsclient.Pools.BroadcastClient(intUserId, kickRes)
	wsclient.Pools.RemoveUserFromGuildPool(intGuildId, intUserId)
	wsclient.Pools.BroadcastGuild(intGuildId, guildRes)
	c.Status(http.StatusNoContent)
}
