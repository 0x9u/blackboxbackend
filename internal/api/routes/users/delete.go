package users

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"

	"github.com/asianchinaboi/backendserver/internal/api/middleware"
	"github.com/asianchinaboi/backendserver/internal/db"
	"github.com/asianchinaboi/backendserver/internal/errors"
	"github.com/asianchinaboi/backendserver/internal/events"
	"github.com/asianchinaboi/backendserver/internal/logger"
	"github.com/asianchinaboi/backendserver/internal/session"
	"github.com/asianchinaboi/backendserver/internal/wsclient"
	"github.com/gin-gonic/gin"
)

//TODO: Replace IN with something else (might be inefficient)

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
	if err := c.ShouldBindJSON(&body); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusBadRequest, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusBadRequest,
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
		if err := tx.Rollback(); err != nil {
			logger.Warn.Printf("unable to rollback error: %v\n", err)
		}
	}() //rollback changes if failed

	guildRows, err := tx.QueryContext(ctx, "SELECT guild_id FROM userguilds WHERE user_id = $1", user.Id)
	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	defer guildRows.Close()

	msgGuildRows, err := tx.QueryContext(ctx, "SELECT DISTINCT guild_id FROM msgs WHERE user_id = $1 ", user.Id)
	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	defer msgGuildRows.Close()

	ownedGuildRows, err := tx.QueryContext(ctx, `DELETE FROM guilds u INNER JOIN userguilds ug ON g.id = ug.guild_id 
	WHERE ug.owner = true AND ug.user_id = $1 RETURNING u.id`, user.Id)
	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	defer ownedGuildRows.Close()

	directGuildRows, err := tx.QueryContext(ctx, "SELECT receiver_id, dm_id FROM userdirectmsgsguild WHERE user_id = $1", user.Id)
	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	defer directGuildRows.Close()

	/*
	if _, err := tx.ExecContext(ctx, "DELETE FROM directmsgsguild dmg WHERE dmg.id IN (SELECT dm_id FROM userdirectmsgsguild udmg WHERE udmg.user_id = $1)", user.Id); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}*/ // no longer needed I think - REMOVE if no bugs

	if _, err := tx.ExecContext(ctx, "DELETE FROM users WHERE id = $1", user.Id); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	fileIds, err := tx.QueryContext(ctx, `DELETE FROM files f 
	WHERE NOT EXISTS (SELECT 1 FROM directmsgfiles dmf WHERE f.id = dmf.file_id) 
	AND NOT EXISTS (SELECT 1 FROM msgfiles mf WHERE f.id = mf.file_id) 
	AND NOT EXISTS (SELECT 1 FROM users u WHERE f.id = u.image_id) 
	AND NOT EXISTS (SELECT 1 FROM guilds g WHERE f.id = g.image_id) RETURNING f.id`, user.Id)
	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	defer fileIds.Close()

	if err := tx.Commit(); err != nil { //commits the transaction
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	for fileIds.Next() {
		var fileId int64
		if err := fileIds.Scan(&fileId); err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
		if err := os.Remove(fmt.Sprintf("uploads/%d.lz4", fileId)); err != nil {
			logger.Warn.Printf("unable to remove file: %v\n", err)
		}
	}

	for guildRows.Next() {
		var guildId int64
		if err := guildRows.Scan(&guildId); err != nil {
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

	for msgGuildRows.Next() {
		var msgId, guildId int64
		if err := msgGuildRows.Scan(&msgId, &guildId); err != nil {
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

	for ownedGuildRows.Next() {
		var guildId int64
		if err := ownedGuildRows.Scan(&guildId); err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
		wsclient.Pools.BroadcastGuild(guildId, wsclient.DataFrame{ //makes the client delete guild
			Op: wsclient.TYPE_DISPATCH,
			Data: events.Guild{
				GuildId: guildId,
			},
			Event: events.DELETE_GUILD,
		})
	}

	for directGuildRows.Next() { //clear dms for other users
		var userId int64
		var dmId int64
		if err := directGuildRows.Scan(&userId, &dmId); err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
		wsclient.Pools.BroadcastGuild(userId, wsclient.DataFrame{
			Op: wsclient.TYPE_DISPATCH,
			Data: events.Dm{
				DmId: dmId,
			},
			Event: events.DELETE_DM,
		})
	}

	wsclient.Pools.DisconnectUserFromClientPool(user.Id)
	c.Status(http.StatusNoContent)
}
