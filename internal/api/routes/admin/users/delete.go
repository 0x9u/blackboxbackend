package users

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"

	"github.com/asianchinaboi/backendserver/internal/api/middleware"
	"github.com/asianchinaboi/backendserver/internal/db"
	"github.com/asianchinaboi/backendserver/internal/errors"
	"github.com/asianchinaboi/backendserver/internal/events"
	"github.com/asianchinaboi/backendserver/internal/logger"
	"github.com/asianchinaboi/backendserver/internal/session"
	"github.com/asianchinaboi/backendserver/internal/wsclient"
	"github.com/gin-gonic/gin"
)

type fileEntity struct {
	Id         int64
	EntityType string
}

type guildEntity struct {
	Id int64
	Dm bool
}

//TODO: Replace IN with something else (might be inefficient)

func Delete(c *gin.Context) {
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

	if !user.Perms.Admin && !user.Perms.Users.Delete {
		logger.Error.Println(errors.ErrNotAuthorised)
		c.JSON(http.StatusForbidden, errors.Body{
			Error:  errors.ErrNotAuthorised.Error(),
			Status: errors.StatusNotAuthorised,
		})
		return
	}

	userId := c.Param("userId")
	if match, err := regexp.MatchString("^[0-9]+$", userId); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	} else if !match {
		logger.Error.Println(errors.ErrRouteParamInvalid)
		c.JSON(http.StatusBadRequest, errors.Body{
			Error:  errors.ErrRouteParamInvalid.Error(),
			Status: errors.StatusRouteParamInvalid,
		})
		return
	}

	var userExists bool

	if err := db.Db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", userId).Scan(&userExists); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	if !userExists {
		logger.Error.Println(errors.ErrUserNotFound)
		c.JSON(http.StatusNotFound, errors.Body{
			Error:  errors.ErrUserNotFound.Error(),
			Status: errors.StatusUserNotFound,
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
	defer tx.Rollback() //rollback changes if failed

	guildIds := []int64{}
	guildRows, err := db.Db.Query("SELECT guild_id FROM userguilds WHERE user_id = $1 AND owner = false", userId)
	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
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
		guildIds = append(guildIds, guildId)
	}
	guildRows.Close()

	files := []fileEntity{}
	fileRows, err := db.Db.Query(`SELECT files.id, files.entity_type FROM files LEFT JOIN msgs ON msgs.id = files.msg_id LEFT JOIN userguilds ON userguilds.guild_id = files.guild_id AND userguilds.owner = true 
		LEFT JOIN users ON users.id = files.user_id WHERE msgs.user_id = $1 OR userguilds.user_id = $1 OR users.id = $1
		`, userId)
	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	for fileRows.Next() {
		var file fileEntity
		if err := fileRows.Scan(&file.Id, &file.EntityType); err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
		files = append(files, file)
	}

	fileRows.Close()

	ownedGuilds := []guildEntity{}
	ownedGuildRows, err := tx.QueryContext(ctx, `DELETE FROM guilds u USING userguilds ug 
		WHERE u.id = ug.guild_id AND ug.owner = true AND ug.user_id = $1 RETURNING u.id, u.dm`, userId)
	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	for ownedGuildRows.Next() {
		var guild guildEntity
		if err := ownedGuildRows.Scan(&guild.Id, &guild.Dm); err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
		ownedGuilds = append(ownedGuilds, guild)
	}
	ownedGuildRows.Close()

	if _, err := tx.ExecContext(ctx, "DELETE FROM users WHERE id = $1", userId); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	if err := tx.Commit(); err != nil { //commits the transaction
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	for _, file := range files {
		if err := os.Remove(fmt.Sprintf("uploads/%s/%d.lz4", file.EntityType, file.Id)); err != nil {
			logger.Warn.Printf("unable to remove file: %v\n", err)
		}
	}

	intUserId, err := strconv.ParseInt(userId, 10, 64)
	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	for _, guildId := range guildIds {
		wsclient.Pools.RemoveUserFromGuildPool(guildId, intUserId)
		wsclient.Pools.BroadcastGuild(guildId, wsclient.DataFrame{
			Op: wsclient.TYPE_DISPATCH,
			Data: events.Msg{
				GuildId: guildId,
				Author: events.User{
					UserId: intUserId,
				},
			},
			Event: events.CLEAR_USER_MESSAGES,
		})
		wsclient.Pools.BroadcastGuild(guildId, wsclient.DataFrame{
			Op: wsclient.TYPE_DISPATCH,
			Data: events.Msg{
				GuildId: guildId,
			},
			Event: events.REMOVE_USER_GUILDLIST,
		})
	}

	for _, ownedGuild := range ownedGuilds {

		if ownedGuild.Dm {
			wsclient.Pools.BroadcastGuild(ownedGuild.Id, wsclient.DataFrame{ //makes the client delete guild
				Op: wsclient.TYPE_DISPATCH,
				Data: events.Dm{
					DmId: ownedGuild.Id,
				},
				Event: events.DELETE_DM,
			})
		} else {
			wsclient.Pools.BroadcastGuild(ownedGuild.Id, wsclient.DataFrame{ //makes the client delete guild
				Op: wsclient.TYPE_DISPATCH,
				Data: events.Guild{
					GuildId: ownedGuild.Id,
				},
				Event: events.DELETE_GUILD,
			})
		}
	}

	wsclient.Pools.DisconnectUserFromClientPool(intUserId)
	c.Status(http.StatusNoContent)
}
