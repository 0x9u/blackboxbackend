package msgs

import (
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/asianchinaboi/backendserver/internal/api/middleware"
	"github.com/asianchinaboi/backendserver/internal/db"
	"github.com/asianchinaboi/backendserver/internal/errors"
	"github.com/asianchinaboi/backendserver/internal/events"
	"github.com/asianchinaboi/backendserver/internal/session"
	"github.com/asianchinaboi/backendserver/internal/wsclient"
	"github.com/gin-gonic/gin"
)

func Typing(c *gin.Context) {
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

	intGuildId, err := strconv.ParseInt(guildId, 10, 64)
	if err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	var inGuild bool
	var isDm bool

	if err := db.Db.QueryRow("SELECT EXISTS (SELECT 1 FROM userguilds WHERE guild_id=$1 AND user_id=$2 AND banned=false), EXISTS (SELECT 1 FROM guilds WHERE id = $1 AND dm = true)", guildId, user.Id).Scan(&inGuild, &isDm); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}
	if !inGuild {
		errors.SendErrorResponse(c, errors.ErrNotInGuild, errors.StatusNotInGuild)
		return
	}
	//get user info
	var username string
	if err := db.Db.QueryRow("SELECT username FROM users WHERE id=$1", user.Id).Scan(&username); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	res := wsclient.DataFrame{
		Op: wsclient.TYPE_DISPATCH,
		Data: events.Typing{
			GuildId: intGuildId,
			Time:    time.Now().UTC(),
			UserInfo: events.User{
				UserId: user.Id,
				Name:   username,
			},
		},
		Event: events.TYPING_START,
	}

	wsclient.Pools.BroadcastGuild(intGuildId, res)
	c.Status(http.StatusNoContent)

}
