package msgs

import (
	"context"
	"database/sql"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/asianchinaboi/backendserver/internal/api/middleware"
	"github.com/asianchinaboi/backendserver/internal/db"
	"github.com/asianchinaboi/backendserver/internal/errors"
	"github.com/asianchinaboi/backendserver/internal/events"
	"github.com/asianchinaboi/backendserver/internal/logger"
	"github.com/asianchinaboi/backendserver/internal/session"
	"github.com/asianchinaboi/backendserver/internal/wsclient"
	"github.com/gin-gonic/gin"
)

// expects msg:content
func Edit(c *gin.Context) {
	user := c.MustGet(middleware.User).(*session.Session)
	if user == nil {
		errors.SendErrorResponse(c, errors.ErrSessionDidntPass, errors.StatusInternalError)
		return
	}

	var isRequestId bool
	msgId := c.Param("msgId") //fix request id bug
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

	var msg events.Msg

	if err := c.ShouldBindJSON(&msg); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusBadRequest)
		return
	}

	if msg.Content == "" {
		errors.SendErrorResponse(c, errors.ErrInvalidDetails, errors.StatusInvalidDetails)
		return
	}

	var isDm bool
	var inGuild bool

	if err := db.Db.QueryRow("SELECT EXISTS (SELECT 1 FROM guilds WHERE id = $1 AND dm = true), EXISTS (SELECT * FROM userguilds WHERE guild_id=$1 AND user_id=$2 AND banned=false)", guildId, user.Id).Scan(&isDm, &inGuild); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	if !inGuild {
		errors.SendErrorResponse(c, errors.ErrNotInGuild, errors.StatusNotInGuild)
		return
	}

	var timestamp time.Time

	//BEGIN TRANSACTION
	ctx := context.Background()
	tx, err := db.Db.BeginTx(ctx, nil)
	if err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}
	defer tx.Rollback() //rollback changes if failed

	if !isRequestId { //vulnerability: isrequestid can be updated by any user

		var msgExists bool
		if err := db.Db.QueryRow("SELECT EXISTS(SELECT 1 FROM msgs WHERE id = $1 AND user_id = $2 AND guild_id=$3)", msgId, user.Id, guildId).Scan(&msgExists); err != nil {
			errors.SendErrorResponse(c, err, errors.StatusInternalError)
			return
		}

		if !msgExists {
			errors.SendErrorResponse(c, errors.ErrMsgNotExist, errors.StatusMsgNotExist)
			return
		}

		//TODO: Replace modified with a trigger
		if err = tx.QueryRowContext(ctx, "UPDATE msgs SET content = $1, modified = now() WHERE id = $2 AND user_id = $3 AND guild_id=$4 RETURNING modified", msg.Content, msgId, user.Id, guildId).Scan(&timestamp); err != nil {
			errors.SendErrorResponse(c, err, errors.StatusInternalError)
			return
		}

		if _, err := tx.ExecContext(ctx, "DELETE FROM msgmentions WHERE msg_id = $1", msgId); err != nil {
			errors.SendErrorResponse(c, err, errors.StatusInternalError)
			return
		}

	} else {
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
	}

	timestamp = time.Now()

	mentions := events.MentionExp.FindAllStringSubmatch(msg.Content, -1)
	logger.Debug.Println("msgcontent:", msg.Content)
	logger.Debug.Println("mentions:", mentions)
	msg.MentionsEveryone = events.MentionEveryoneExp.MatchString(msg.Content)
	msg.Mentions = make([]events.User, 0, len(mentions))

	if len(mentions) > 0 {
		logger.Debug.Println("mentions found")
		seen := make(map[int64]bool)
		for _, mention := range mentions {
			mentionUserId, err := strconv.ParseInt(mention[1], 10, 64)
			if err != nil {
				errors.SendErrorResponse(c, err, errors.StatusInternalError)
				return
			}
			if seen[mentionUserId] {
				continue
			}
			seen[mentionUserId] = true

			var mentionUser events.User
			mentionUser.UserId = mentionUserId
			if err := db.Db.QueryRow("SELECT username FROM users WHERE id = $1", mentionUserId).Scan(&mentionUser.Name); err != nil && err != sql.ErrNoRows {
				errors.SendErrorResponse(c, err, errors.StatusInternalError)
				return
			} else if err == sql.ErrNoRows {
				errors.SendErrorResponse(c, errors.ErrUserNotFound, errors.StatusUserNotFound)
				return
			}
			if !isRequestId {
				logger.Debug.Println("inserting mention", mentionUserId, msgId)
				if _, err := tx.ExecContext(ctx, "INSERT INTO msgmentions (msg_id, user_id) VALUES ($1, $2)", msgId, mentionUserId); err != nil {
					errors.SendErrorResponse(c, err, errors.StatusInternalError)
					return
				}
			}

			msg.Mentions = append(msg.Mentions, mentionUser)
		}
	}

	if err := tx.Commit(); err != nil { //commits the transaction
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
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

	var statusMessage string

	if isDm {
		statusMessage = events.UPDATE_DM_MESSAGE
	} else {
		statusMessage = events.UPDATE_GUILD_MESSAGE
	}

	res := wsclient.DataFrame{
		Op: wsclient.TYPE_DISPATCH,
		Data: events.Msg{
			MsgId:            intMsgId,
			GuildId:          intGuildId,
			Content:          msg.Content,
			RequestId:        requestId,
			Mentions:         msg.Mentions,
			MentionsEveryone: msg.MentionsEveryone,
			Modified:         timestamp,
			Author: events.User{
				UserId: user.Id,
			},
		},
		Event: statusMessage,
	}
	wsclient.Pools.BroadcastGuild(intGuildId, res)
	c.Status(http.StatusNoContent)
}
