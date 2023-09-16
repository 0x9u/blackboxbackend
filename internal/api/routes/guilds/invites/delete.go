package invites

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

func Delete(c *gin.Context) {
	user := c.MustGet(middleware.User).(*session.Session)
	if user == nil {
		errors.SendErrorResponse(c, errors.ErrSessionDidntPass, errors.StatusInternalError)
		return
	}

	invite := c.Param("invite")
	if match, err := regexp.MatchString(`^\w+$`, invite); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	} else if !match {
		errors.SendErrorResponse(c, errors.ErrRouteParamInvalid, errors.StatusRouteParamInvalid)
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

	var hasAuth bool
	var inviteValid bool
	var isDm bool
	if err := db.Db.QueryRow("SELECT EXISTS (SELECT 1 FROM invites WHERE invite = $1 AND guild_id=$2), EXISTS (SELECT 1 FROM guilds WHERE id = $2 AND dm = true)", invite, guildId).Scan(&inviteValid, &isDm); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	if isDm {
		errors.SendErrorResponse(c, errors.ErrGuildIsDm, errors.StatusGuildIsDm)
		return
	}

	if !inviteValid {
		errors.SendErrorResponse(c, errors.ErrInvalidInvite, errors.StatusInvalidInvite)
		return
	}

	if err := db.Db.QueryRow("SELECT EXISTS (SELECT 1 FROM userguilds WHERE guild_id=$1 AND user_id=$2 AND (owner=true OR admin=true))", guildId, user.Id).Scan(&hasAuth); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}
	if !hasAuth {
		errors.SendErrorResponse(c, errors.ErrNotGuildAuthorised, errors.StatusNotGuildAuthorised)
		return
	}

	if _, err := db.Db.Exec("DELETE FROM invites WHERE invite=$1", invite); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	intGuildId, err := strconv.ParseInt(guildId, 10, 64)
	if err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	res := wsclient.DataFrame{
		Op: wsclient.TYPE_DISPATCH,
		Data: events.Invite{
			Invite:  invite,
			GuildId: intGuildId,
		},
		Event: events.DELETE_INVITE,
	}
	wsclient.Pools.BroadcastGuild(intGuildId, res)
	c.Status(http.StatusNoContent)
}
