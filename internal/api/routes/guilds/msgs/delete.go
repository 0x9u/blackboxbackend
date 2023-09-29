package msgs

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/asianchinaboi/backendserver/internal/api/middleware"
	"github.com/asianchinaboi/backendserver/internal/db"
	"github.com/asianchinaboi/backendserver/internal/errors"
	"github.com/asianchinaboi/backendserver/internal/events"
	"github.com/asianchinaboi/backendserver/internal/session"
	"github.com/asianchinaboi/backendserver/internal/wsclient"
	"github.com/gin-gonic/gin"
)

func Delete(c *gin.Context) { //deletes message
	user := c.MustGet(middleware.User).(*session.Session)
	if user == nil {
		errors.SendErrorResponse(c, errors.ErrSessionDidntPass, errors.StatusInternalError)
		return
	}

	var isRequestId bool
	msgId := c.Param("msgId")
	if match, err := regexp.MatchString("^[0-9]+$", msgId); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	} else if !match {
		if match, err := regexp.MatchString("^[0-9]+-[0-9]+$", msgId); err != nil {
			errors.SendErrorResponse(c, err, errors.StatusInternalError)
			return
		} else if !match {
			errors.SendErrorResponse(c, errors.ErrRouteParamInvalid, errors.StatusRouteParamInvalid)
			return
		}
		isRequestId = true
	}

	guildId := c.Param("guildId")
	if match, err := regexp.MatchString("^[0-9]+$", guildId); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	} else if !match {
		errors.SendErrorResponse(c, errors.ErrRouteParamInvalid, errors.StatusRouteParamInvalid)
		return
	}

	intGuildId, err := strconv.ParseInt(guildId, 10, 64)
	if err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	var hasAuth bool
	var isDm bool
	if err := db.Db.QueryRow("SELECT EXISTS (SELECT 1 FROM userguilds WHERE guild_id=$1 AND user_id=$2 AND owner=true OR admin=true), EXISTS (SELECT 1 FROM guilds WHERE id = $1 AND dm = true)", guildId, user.Id).Scan(&hasAuth, &isDm); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
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

	var fileRows *sql.Rows

	//theortically if the user did not have chat saved and ran this request it would still work
	if isRequestId {
		requestIdParts := strings.Split(msgId, "-") //should be protected by two in length from regex
		authorId := requestIdParts[0]
		intAuthorId, err := strconv.ParseInt(authorId, 10, 64)
		if err != nil {
			errors.SendErrorResponse(c, err, errors.StatusInternalError)
			return
		}
		if intAuthorId != user.Id { //if the user is not the author of the message
			errors.SendErrorResponse(c, errors.ErrMsgNotExist, errors.StatusMsgNotExist)
			return
		}
	} else {
		var isAuthor bool
		if err := db.Db.QueryRow("SELECT EXISTS (SELECT 1 FROM msgs WHERE id = $1 AND user_id = $2)", msgId, user.Id).Scan(&isAuthor); err != nil {
			errors.SendErrorResponse(c, err, errors.StatusInternalError)
			return
		}
		if !hasAuth && !isAuthor {
			errors.SendErrorResponse(c, errors.ErrNotGuildAuthorised, errors.StatusNotGuildAuthorised)
			return
		}
		var msgExists bool
		if err := db.Db.QueryRow("SELECT EXISTS(SELECT 1 FROM msgs WHERE id = $1 AND guild_id=$2)", msgId, guildId).Scan(&msgExists); err != nil {
			errors.SendErrorResponse(c, err, errors.StatusInternalError)
			return
		}

		if !msgExists {
			errors.SendErrorResponse(c, errors.ErrMsgNotExist, errors.StatusMsgNotExist)
			return
		}

		//delete files associated with msg
		fileRows, err = db.Db.Query("SELECT id FROM files WHERE msg_id = $1", msgId)
		if err != nil {
			errors.SendErrorResponse(c, err, errors.StatusInternalError)
			return
		}
		defer fileRows.Close()

		if _, err = tx.ExecContext(ctx, "DELETE FROM msgs where id = $1 AND guild_id = $2 AND user_id = $3", msgId, guildId, user.Id); err != nil {
			errors.SendErrorResponse(c, err, errors.StatusInternalError)
			return
		}
	}

	var requestId string
	var intMsgId int64
	if isRequestId {
		requestId = msgId
		requestIdParts := strings.Split(msgId, "-")
		intMsgId, err = strconv.ParseInt(requestIdParts[1], 10, 64)
		if err != nil {
			errors.SendErrorResponse(c, err, errors.StatusInternalError)
			return
		}
	} else {
		requestId = "" //there for readabilty
		intMsgId, err = strconv.ParseInt(msgId, 10, 64)
		if err != nil {
			errors.SendErrorResponse(c, err, errors.StatusInternalError)
			return
		}
	}

	if err := tx.Commit(); err != nil { //commits the transaction
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	if fileRows != nil { //if there are files to delete
		for fileRows.Next() {
			var fileId int64
			if err := fileRows.Scan(&fileId); err != nil {
				errors.SendErrorResponse(c, err, errors.StatusInternalError)
				return
			}
			if err := os.Remove(fmt.Sprintf("uploads/msg/%d.lz4", fileId)); err != nil {
				errors.SendErrorResponse(c, err, errors.StatusInternalError)
				return
			}
		}
	}

	res := wsclient.DataFrame{
		Op: wsclient.TYPE_DISPATCH,
		Data: events.Msg{
			MsgId:     intMsgId,
			GuildId:   intGuildId,
			RequestId: requestId,
		},
		Event: events.MESSAGE_DELETE,
	}
	wsclient.Pools.BroadcastGuild(intGuildId, res)
	c.Status(http.StatusNoContent)
}

func Clear(c *gin.Context) {
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

	intGuildId, err := strconv.ParseInt(guildId, 10, 64)
	if err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	var hasAuth bool
	var isDm bool
	if err := db.Db.QueryRow("SELECT EXISTS (SELECT 1 FROM userguilds WHERE guild_id=$1 AND user_id=$2 AND (owner=true OR admin=true)), EXISTS (SELECT 1 FROM userguilds WHERE guild_id=$1 AND dm=true)", guildId, user.Id).Scan(&hasAuth, &isDm); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}
	if !hasAuth {
		errors.SendErrorResponse(c, errors.ErrNotGuildAuthorised, errors.StatusNotGuildAuthorised)
		return
	}

	fileRows, err := db.Db.Query("SELECT files.id FROM files INNER JOIN msgs ON msg.id=files.msg_id WHERE msgs.guild_id = $1", guildId)
	if err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}
	defer fileRows.Close()

	if _, err := db.Db.Exec("DELETE FROM msgs WHERE guild_id = $1", guildId); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	if fileRows != nil { //if there are files to delete
		for fileRows.Next() {
			var fileId int64
			if err := fileRows.Scan(&fileId); err != nil {
				errors.SendErrorResponse(c, err, errors.StatusInternalError)
				return
			}
			if err := os.Remove(fmt.Sprintf("uploads/msg/%d.lz4", fileId)); err != nil {
				errors.SendErrorResponse(c, err, errors.StatusInternalError)
				return
			}
		}
	}
	res := wsclient.DataFrame{
		Op: wsclient.TYPE_DISPATCH,
		Data: events.Msg{
			GuildId: intGuildId,
		},
		Event: events.MESSAGES_GUILD_CLEAR,
	}
	wsclient.Pools.BroadcastGuild(intGuildId, res)
	c.Status(http.StatusNoContent)
}
