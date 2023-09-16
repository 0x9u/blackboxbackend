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
	"github.com/gin-gonic/gin"
)

func Get(c *gin.Context) {
	user := c.MustGet(middleware.User).(*session.Session)
	if user == nil {
		errors.SendErrorResponse(c, errors.ErrSessionDidntPass, errors.StatusInternalError)
		return
	}
	guildId := c.Param("guildId")
	if match, err := regexp.MatchString("^[0-9]+$", guildId); err != nil {
		errors.SendErrorResponse(c, errors.ErrSessionDidntPass, errors.StatusInternalError)
		return
	} else if !match {
		errors.SendErrorResponse(c, errors.ErrRouteParamInvalid, errors.StatusRouteParamInvalid)
		return
	}

	var inGuild bool
	var isDm bool
	if err := db.Db.QueryRow("SELECT EXISTS (SELECT * FROM userguilds WHERE guild_id=$1 AND user_id=$2), EXISTS (SELECT 1 FROM guilds WHERE id = $1 AND dm = true)", guildId, user.Id).Scan(&inGuild, &isDm); err != nil {
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
	rows, err := db.Db.Query("SELECT invite FROM invites WHERE guild_id=$1", guildId)
	if err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}
	inviteList := []events.Invite{}
	intGuildId, err := strconv.ParseInt(guildId, 10, 64)
	if err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}
	for rows.Next() {
		var invite events.Invite
		if err := rows.Scan(&invite.Invite); err != nil {
			errors.SendErrorResponse(c, err, errors.StatusInternalError)
			return
		}
		invite.GuildId = intGuildId
		inviteList = append(inviteList, invite)
	}
	c.JSON(http.StatusOK, inviteList)
}
