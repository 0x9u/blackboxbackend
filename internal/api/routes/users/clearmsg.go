package users

import (
	"context"
	"net/http"

	"github.com/asianchinaboi/backendserver/internal/api/middleware"
	"github.com/asianchinaboi/backendserver/internal/db"
	"github.com/asianchinaboi/backendserver/internal/errors"
	"github.com/asianchinaboi/backendserver/internal/events"
	"github.com/asianchinaboi/backendserver/internal/logger"
	"github.com/asianchinaboi/backendserver/internal/session"
	"github.com/asianchinaboi/backendserver/internal/wsclient"
	"github.com/gin-gonic/gin"
)

func clearUserMsg(c *gin.Context) {
	user := c.MustGet(middleware.User).(*session.Session)
	if user == nil {
		logger.Error.Println("user token not sent in data")
		c.JSON(http.StatusInternalServerError,
			errors.Body{
				Error:  errors.ErrSessionDidntPass.Error(),
				Status: errors.StatusInternalError,
			})
		return
	}

	guildRows, err := db.Db.Query("SELECT DISTINCT guild_id FROM msgs WHERE user_id = $1", user.Id)

	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	defer guildRows.Close()

	//BEGIN TRANSACTION
	ctx := context.Background()
	tx, err := db.Db.BeginTx(ctx, nil)
	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	defer func() {
		if err != nil {
			if err := tx.Rollback(); err != nil {
				logger.Warn.Printf("unable to rollback error: %v\n", err)
			}
		}
	}() //rollback changes if failed

	if _, err = tx.ExecContext(ctx, "DELETE FROM msgs WHERE user_id = $1", user.Id); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	if _, err = tx.ExecContext(ctx, "DELETE FROM directmsgs WHERE user_id = $1", user.Id); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	dmRows, err := db.Db.Query("SELECT dm_id, user_id FROM directmsgs WHERE dm_id IN (SELECT dm_id FROM directmsgs WHERE user_id = $1) AND user_id != $1", user.Id)

	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	defer dmRows.Close()

	if err = tx.Commit(); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	for guildRows.Next() {
		var guildId int64
		err = guildRows.Scan(&guildId)
		if err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
		clearMsg := events.Msg{
			Author: events.User{
				UserId: user.Id,
			},
			GuildId: guildId,
		}
		res := wsclient.DataFrame{
			Op:    wsclient.TYPE_DISPATCH,
			Data:  clearMsg,
			Event: events.CLEAR_USER_MESSAGES,
		}
		if err := wsclient.Pools.BroadcastGuild(guildId, res); err != nil && err != errors.ErrGuildPoolNotExist {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
	}

	for dmRows.Next() {
		var dmId int64
		var otherUser int64
		err = dmRows.Scan(&dmId, &otherUser)
		if err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
		clearMsg := events.Msg{
			Author: events.User{
				UserId: user.Id,
			},
			DmId: dmId,
		}
		res := wsclient.DataFrame{
			Op:    wsclient.TYPE_DISPATCH,
			Data:  clearMsg,
			Event: events.CLEAR_USER_DM_MESSAGES,
		}
		if err := wsclient.Pools.BroadcastClient(otherUser, res); err != nil && err != errors.ErrUserClientNotExist {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
		if err := wsclient.Pools.BroadcastClient(user.Id, res); err != nil && err != errors.ErrUserClientNotExist {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
	}

	c.Status(http.StatusNoContent)
}
