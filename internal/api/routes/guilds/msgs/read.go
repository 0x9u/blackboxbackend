package msgs

import (
	"net/http"
	"regexp"
	"time"

	"github.com/asianchinaboi/backendserver/internal/api/middleware"
	"github.com/asianchinaboi/backendserver/internal/db"
	"github.com/asianchinaboi/backendserver/internal/errors"
	"github.com/asianchinaboi/backendserver/internal/session"
	"github.com/gin-gonic/gin"
)

// figure out alternative ways to acknowledge read messages
func Read(c *gin.Context) {
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

	var inGuild bool

	if err := db.Db.QueryRow("SELECT EXISTS (SELECT * FROM guilds WHERE id=$1)", guildId).Scan(&inGuild); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	if !inGuild {
		errors.SendErrorResponse(c, errors.ErrNotInGuild, errors.StatusNotInGuild)
		return
	}

	var lastMsgId int
	if err := db.Db.QueryRow("SELECT MAX(id) FROM msgs WHERE guild_id = $1", guildId).Scan(&lastMsgId); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	var lastMsgTime time.Time
	if err := db.Db.QueryRow("SELECT created FROM msgs WHERE id = $1", lastMsgId).Scan(&lastMsgTime); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	if _, err := db.Db.Exec("UPDATE unreadmsgs SET msg_id = $3, time = $4 WHERE user_id = $2 AND guild_id = $1", guildId, user.Id, lastMsgId, lastMsgTime); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}
	c.Status(http.StatusNoContent)
}
