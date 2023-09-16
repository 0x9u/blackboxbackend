package guilds

import (
	"context"
	"database/sql"
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

func deleteGuild(c *gin.Context) {
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

	var isOwner bool
	var isDm bool
	if err := db.Db.QueryRow("SELECT EXISTS (SELECT 1 FROM userguilds WHERE guild_id=$1 AND user_id=$2 AND owner=true), EXISTS (SELECT 1 FROM guilds WHERE id = $1 AND dm = true)", guildId, user.Id).Scan(&isOwner, &isDm); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}
	if isDm {
		errors.SendErrorResponse(c, errors.ErrGuildIsDm, errors.StatusGuildIsDm)
		return
	}
	if !isOwner {
		errors.SendErrorResponse(c, errors.ErrNotGuildAuthorised, errors.StatusNotGuildAuthorised)
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

	var guildImageId int64
	if err := tx.QueryRowContext(ctx, "SELECT id FROM files WHERE guild_id = $1", guildId).Scan(&guildImageId); err != nil && err != sql.ErrNoRows {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	} else if err == sql.ErrNoRows {
		guildImageId = -1
	}

	fileIds, err := tx.QueryContext(ctx, `SELECT f.id FROM files f INNER JOIN msgs ON msgs.id = f.msg_id WHERE msgs.guild_id = $1`, guildId)
	if err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}
	var fileIdsToDelete []int64

	for fileIds.Next() {
		var fileId int64
		if err := fileIds.Scan(&fileId); err != nil {
			errors.SendErrorResponse(c, err, errors.StatusInternalError)
			return
		}
		fileIdsToDelete = append(fileIdsToDelete, fileId)
	}

	fileIds.Close()

	if _, err := tx.ExecContext(ctx, "DELETE FROM guilds WHERE id=$1", guildId); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	intGuildId, err := strconv.ParseInt(guildId, 10, 64)
	if err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	if err := tx.Commit(); err != nil { //commits the transaction
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	if guildImageId != -1 {
		if err := os.Remove(fmt.Sprintf("uploads/guild/%d.lz4", guildImageId)); err != nil {
			logger.Warn.Printf("unable to remove file: %v\n", err)
		}
	}

	for _, fileId := range fileIdsToDelete {
		if err := os.Remove(fmt.Sprintf("uploads/msg/%d.lz4", fileId)); err != nil {
			logger.Warn.Printf("unable to remove file: %v\n", err)
		}
	}

	res := wsclient.DataFrame{
		Op: wsclient.TYPE_DISPATCH,
		Data: events.Guild{
			GuildId: intGuildId,
		},
		Event: events.DELETE_GUILD,
	}
	wsclient.Pools.BroadcastGuild(intGuildId, res) // kick everyone out of the guild
	c.Status(http.StatusNoContent)
}
