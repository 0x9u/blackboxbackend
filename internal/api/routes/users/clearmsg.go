package users

import (
	"context"
	"net/http"

	"github.com/asianchinaboi/backendserver/internal/api/middleware"
	"github.com/asianchinaboi/backendserver/internal/db"
	"github.com/asianchinaboi/backendserver/internal/errors"
	"github.com/asianchinaboi/backendserver/internal/events"
	"github.com/asianchinaboi/backendserver/internal/session"
	"github.com/asianchinaboi/backendserver/internal/wsclient"
	"github.com/gin-gonic/gin"
)

func clearUserMsg(c *gin.Context) {
	user := c.MustGet(middleware.User).(*session.Session)
	if user == nil {
		errors.SendErrorResponse(c, errors.ErrSessionDidntPass, errors.StatusInternalError)
		return
	}

	guildRows, err := db.Db.Query("SELECT DISTINCT msgs.guild_id, guilds.dm FROM msgs INNER JOIN guilds ON msgs.guild_id = guilds.id WHERE user_id = $1", user.Id)

	if err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	defer guildRows.Close()

	//BEGIN TRANSACTION
	ctx := context.Background()
	tx, err := db.Db.BeginTx(ctx, nil)
	if err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}
	defer tx.Rollback() //rollback changes if failed

	if _, err = tx.ExecContext(ctx, "DELETE FROM msgs WHERE user_id = $1", user.Id); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	if err = tx.Commit(); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	for guildRows.Next() {
		var guildId int64
		var isDm bool
		err = guildRows.Scan(&guildId, &isDm)
		if err != nil {
			errors.SendErrorResponse(c, err, errors.StatusInternalError)
			return
		}
		clearMsg := events.Msg{
			Author: events.User{
				UserId: user.Id,
			},
			GuildId: guildId,
		}
		var statusMessage string
		if isDm {
			statusMessage = events.CLEAR_USER_DM_MESSAGES
		} else {
			statusMessage = events.CLEAR_USER_MESSAGES
		}
		res := wsclient.DataFrame{
			Op:    wsclient.TYPE_DISPATCH,
			Data:  clearMsg,
			Event: statusMessage,
		}
		if err := wsclient.Pools.BroadcastGuild(guildId, res); err != nil && err != errors.ErrGuildPoolNotExist {
			errors.SendErrorResponse(c, err, errors.StatusInternalError)
			return
		}
	}

	c.Status(http.StatusNoContent)
}
