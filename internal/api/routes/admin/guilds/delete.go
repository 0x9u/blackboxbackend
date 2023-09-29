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

func Delete(c *gin.Context) {
	user := c.MustGet(middleware.User).(*session.Session)
	if user == nil {
		errors.SendErrorResponse(c, errors.ErrSessionDidntPass, errors.StatusInternalError)
		return
	}
	if !user.Perms.Admin && !user.Perms.Guilds.Delete {
		errors.SendErrorResponse(c, errors.ErrNotAuthorised, errors.StatusNotAuthorised)
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

	//BEGIN TRANSACTION
	ctx := context.Background()
	tx, err := db.Db.BeginTx(ctx, nil)
	if err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}
	defer tx.Rollback()

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
		Event: events.GUILD_DELETE,
	}

	wsclient.Pools.BroadcastGuild(intGuildId, res) // kick everyone out of the guild
	c.Status(http.StatusNoContent)
}
