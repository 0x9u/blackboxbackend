package invites

import (
	"net/http"
	"regexp"
	"strconv"

	"github.com/asianchinaboi/backendserver/internal/api/middleware"
	"github.com/asianchinaboi/backendserver/internal/config"
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

	var count int
	var inGuild bool
	var isDm bool
	if err := db.Db.QueryRow("SELECT EXISTS(SELECT 1 FROM userguilds WHERE user_id=$1 AND guild_id = $2) ,EXISTS (SELECT 1 FROM guilds WHERE id = $2 AND dm = true) ", user.Id, guildId).Scan(&inGuild, &isDm); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	if isDm {
		errors.SendErrorResponse(c, errors.ErrGuildIsDm, errors.StatusGuildIsDm)
		return
	}

	if !inGuild {
		errors.SendErrorResponse(c, errors.ErrNotInGuild, errors.StatusNotInGuild)
		return
	}
	if err := db.Db.QueryRow("SELECT COUNT(*) FROM invites WHERE guild_id=$1", guildId).Scan(&count); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}
	if count > config.Config.Guild.MaxInvites {
		errors.SendErrorResponse(c, errors.ErrInviteLimitReached, errors.StatusInviteLimitReached)
		return
	}

	invite := session.GenerateRandString(10)

	if _, err := db.Db.Exec("INSERT INTO invites (invite, guild_id) VALUES ($1, $2)", invite, guildId); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	intGuildId, err := strconv.ParseInt(guildId, 10, 64)
	if err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	inviteBody := events.Invite{
		Invite:  invite,
		GuildId: intGuildId,
	}
	res := wsclient.DataFrame{
		Op:    wsclient.TYPE_DISPATCH,
		Data:  inviteBody,
		Event: events.INVITE_CREATE,
	}

	wsclient.Pools.BroadcastGuild(intGuildId, res)
	c.JSON(http.StatusOK, inviteBody)
}
