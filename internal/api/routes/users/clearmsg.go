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

	rows, err := db.Db.Query("SELECT DISTINCT guild_id FROM messages WHERE user_id = $1", user.Id)

	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	defer rows.Close()

	if _, err = db.Db.Exec("DELETE FROM messages WHERE user_id = $1", user.Id); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	for rows.Next() {
		var guildId int
		err = rows.Scan(&guildId)
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
	c.Status(http.StatusOK)
}
