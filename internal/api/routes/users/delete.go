package users

import (
	"context"
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

type fileEntity struct {
	Id         int64
	EntityType string
}

type guildEntity struct {
	Id int64
	Dm bool
}

//TODO: Replace IN with something else (might be inefficient)

func userDelete(c *gin.Context) {
	user := c.MustGet(middleware.User).(*session.Session)
	if user == nil {
		errors.SendErrorResponse(c, errors.ErrSessionDidntPass, errors.StatusInternalError)
		return
	}

	var body events.User
	if err := c.ShouldBindJSON(&body); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusBadRequest)
		return
	}

	var userHashedPass string
	if err := db.Db.QueryRow("SELECT password FROM users WHERE id = $1", user.Id).Scan(&userHashedPass); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	if correctPass := comparePasswords(body.Password, userHashedPass); !correctPass {
		errors.SendErrorResponse(c, errors.ErrInvalidPass, errors.StatusInvalidPass)
		return
	}

	//BEGIN TRANSACTION
	ctx := context.Background()
	tx, err := db.Db.BeginTx(ctx, nil)
	if err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}
	defer tx.Rollback() //rollback changes if failed

	guildIds := []int64{}
	guildRows, err := db.Db.Query("SELECT guild_id FROM userguilds WHERE user_id = $1 AND owner = false", user.Id)
	if err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}
	for guildRows.Next() {
		var guildId int64
		if err := guildRows.Scan(&guildId); err != nil {
			errors.SendErrorResponse(c, err, errors.StatusInternalError)
			return
		}
		guildIds = append(guildIds, guildId)
	}
	guildRows.Close()

	files := []fileEntity{}
	fileRows, err := db.Db.Query(`SELECT files.id, files.entity_type FROM files LEFT JOIN msgs ON msgs.id = files.msg_id LEFT JOIN userguilds ON userguilds.guild_id = files.guild_id AND userguilds.owner = true 
	LEFT JOIN users ON users.id = files.user_id WHERE msgs.user_id = $1 OR userguilds.user_id = $1 OR users.id = $1
	`, user.Id)
	if err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	for fileRows.Next() {
		var file fileEntity
		if err := fileRows.Scan(&file.Id, &file.EntityType); err != nil {
			errors.SendErrorResponse(c, err, errors.StatusInternalError)
			return
		}
		files = append(files, file)
	}

	fileRows.Close()

	ownedGuilds := []guildEntity{}
	ownedGuildRows, err := tx.QueryContext(ctx, `DELETE FROM guilds u USING userguilds ug 
	WHERE u.id = ug.guild_id AND ug.owner = true AND ug.user_id = $1 RETURNING u.id, u.dm`, user.Id)
	if err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}
	for ownedGuildRows.Next() {
		var guild guildEntity
		if err := ownedGuildRows.Scan(&guild.Id, &guild.Dm); err != nil {
			errors.SendErrorResponse(c, err, errors.StatusInternalError)
			return
		}
		ownedGuilds = append(ownedGuilds, guild)
	}
	ownedGuildRows.Close()

	if _, err := tx.ExecContext(ctx, "DELETE FROM users WHERE id = $1", user.Id); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	if err := tx.Commit(); err != nil { //commits the transaction
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	for _, file := range files {
		if err := os.Remove(fmt.Sprintf("uploads/%s/%d.lz4", file.EntityType, file.Id)); err != nil {
			logger.Warn.Printf("unable to remove file: %v\n", err)
		}
	}

	for _, guildId := range guildIds {
		wsclient.Pools.RemoveUserFromGuildPool(guildId, user.Id)
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

	wsclient.Pools.DisconnectUserFromClientPool(user.Id)
	c.Status(http.StatusNoContent)
}
