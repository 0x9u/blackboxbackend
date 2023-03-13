package users

import (
	"database/sql"
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

func userDelete(c *gin.Context) {
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

	var body events.User
	if err := c.ShouldBindJSON(&user); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusBadRequest, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusBadJSON,
		})
		return
	}

	var userHashedPass string
	if err := db.Db.QueryRow("SELECT password FROM users WHERE username = $1", body.Name).Scan(&userHashedPass); err != nil && err != sql.ErrNoRows {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	} else if err != nil {
		logger.Error.Println(errors.ErrUserNotFound)
		c.JSON(http.StatusNotFound, errors.Body{
			Error:  errors.ErrUserNotFound.Error(),
			Status: errors.StatusUserNotFound,
		})
		return
	}
	if correctPass := comparePasswords(body.Password, userHashedPass); !correctPass {
		logger.Error.Println(errors.ErrInvalidPass)
		c.JSON(http.StatusForbidden, errors.Body{
			Error:  errors.ErrInvalidPass.Error(),
			Status: errors.StatusInvalidPass,
		})
		return
	}

	if _, err := db.Db.Exec("DELETE FROM tokens WHERE user_id = $1", user.Id); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	//no need to remove user from guild since we are disconnecting the user anyways
	rows, err := db.Db.Query("WITH guildIds AS (DELETE FROM userguilds WHERE user_id = $1 AND owner = false) SELECT DISTINCT guild_id FROM guildIds", user.Id)
	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	for rows.Next() {
		var guildId int64
		if err := rows.Scan(&guildId); err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}

		wsclient.Pools.RemoveUserFromGuildPool(guildId, user.Id)
		wsclient.Pools.BroadcastGuild(guildId, wsclient.DataFrame{
			Op: wsclient.TYPE_DISPATCH,
			Data: events.Msg{
				GuildId: guildId,
			},
			Event: events.REMOVE_USER_GUILDLIST,
		})
	}

	rows, err = db.Db.Query("WITH guildIds AS (DELETE FROM messages WHERE user_id = $1 RETURNING guild_id) SELECT DISTINCT guild_id FROM guildIds ", user.Id)
	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	for rows.Next() {
		var msgId, guildId int64
		if err := rows.Scan(&msgId, &guildId); err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
		wsclient.Pools.BroadcastGuild(guildId, wsclient.DataFrame{
			Op: wsclient.TYPE_DISPATCH,
			Data: events.Msg{
				GuildId: guildId,
				Author: events.User{
					UserId: user.Id,
				},
			},
			Event: events.CLEAR_USER_MESSAGES,
		})
	}

	//remove guilds owned by user
	if _, err := db.Db.Exec("DELETE FROM messages WHERE guild_id IN (SELECT guild_id FROM userguilds WHERE AND user_id = $1)", user.Id); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	if _, err := db.Db.Exec("DELETE FROM invites WHERE guild_id IN (SELECT guild_id FROM userguilds WHERE AND user_id = $1)", user.Id); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	//probs slow
	//dont need to check if owner true since we already deleted all guild association that isnt owner

	//this crazy query deletes all guilds that the user owns and deletes all guild user association for all users including owner
	rows, err = db.Db.Query(`
	DELETE FROM guilds WHERE id IN 
		( DELETE FROM userguilds WHERE guild_id IN ( 
			SELECT guild_id FROM userguilds WHERE user_id = $1 )
	) RETURNING id`, user.Id) //id is all unique from guilds no need for with clause
	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	for rows.Next() {
		var guildId int64
		if err := rows.Scan(&guildId); err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
		wsclient.Pools.BroadcastGuild(guildId, wsclient.DataFrame{ //makes the client delete guild
			Op: wsclient.TYPE_DISPATCH,
			Data: events.Msg{
				GuildId: guildId,
			},
			Event: events.DELETE_GUILD,
		})
	}

	if _, err := db.Db.Exec("DELETE FROM users WHERE id = $1", user.Id); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	//delete roles associated with user
	if _, err := db.Db.Exec("DELETE FROM userroles WHERE user_id = $1", user.Id); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	wsclient.Pools.DisconnectUserFromClientPool(user.Id)
	c.Status(http.StatusNoContent)
}
