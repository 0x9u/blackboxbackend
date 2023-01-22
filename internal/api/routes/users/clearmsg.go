package users

import (
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

	if _, err = db.Db.Exec("DELETE FROM msgs WHERE user_id = $1", user.Id); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	userRows, err := db.Db.Query("SELECT DISTINCT receiver_id FROM directmsgs WHERE sender_id = $1", user.Id)

	for guildRows.Next() {
		var guildId int
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

	for userRows.Next() {
		var receiverId int
		err = userRows.Scan(&receiverId)
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
			UserId: receiverId,
		}
		res := wsclient.DataFrame{
			Op:    wsclient.TYPE_DISPATCH,
			Data:  clearMsg,
			Event: events.CLEAR_USER_DM_MESSAGES,
		}
		if err := wsclient.Pools.BroadcastClient(receiverId, res); err != nil && err != errors.ErrUserClientNotExist {
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
