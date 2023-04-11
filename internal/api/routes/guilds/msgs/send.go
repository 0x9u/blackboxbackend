package msgs

import (
	"context"
	"database/sql"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/asianchinaboi/backendserver/internal/api/middleware"
	"github.com/asianchinaboi/backendserver/internal/config"
	"github.com/asianchinaboi/backendserver/internal/db"
	"github.com/asianchinaboi/backendserver/internal/errors"
	"github.com/asianchinaboi/backendserver/internal/events"
	"github.com/asianchinaboi/backendserver/internal/logger"
	"github.com/asianchinaboi/backendserver/internal/session"
	"github.com/asianchinaboi/backendserver/internal/uid"
	"github.com/asianchinaboi/backendserver/internal/wsclient"
	"github.com/gin-gonic/gin"
	"github.com/pierrec/lz4/v4"
)

// expects
// content : string
func Send(c *gin.Context) {
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

	guildId := c.Param("guildId")
	if match, err := regexp.MatchString("^[0-9]+$", guildId); err != nil {
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

	intGuildId, err := strconv.ParseInt(guildId, 10, 64)
	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	var msg events.Msg
	if err := c.ShouldBindJSON(&msg); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusBadRequest, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusBadRequest,
		})
		return
	}

	//send msg to database
	//broadcast msg to all connections to websocket
	var inGuild bool
	if err := db.Db.QueryRow("SELECT EXISTS (SELECT * FROM userguilds WHERE guild_id=$1 AND user_id=$2 AND banned=false)", guildId, user.Id).Scan(&inGuild); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	if !inGuild {
		logger.Error.Println(errors.ErrNotInGuild)
		c.JSON(http.StatusForbidden, errors.Body{
			Error:  errors.ErrNotInGuild.Error(),
			Status: errors.StatusNotInGuild,
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

	msg.Content = strings.TrimSpace(msg.Content)
	//screw off html
	msg.Content = html.EscapeString(msg.Content) //prevents xss attacks
	msg.Created = time.Now().Unix()
	msg.MsgId = uid.Snowflake.Generate().Int64()
	//check if attachments uploaded

	form, err := c.MultipartForm()
	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusBadRequest, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusBadRequest,
		})
		return
	}

	files := form.File["files[]"]
	msg.Attachments = []events.Attachment{}
	fileIds := []int64{}
	fileSucessful := false
	defer func() {
		if !fileSucessful {
			for _, id := range fileIds {
				if err := os.Remove(fmt.Sprintf("uploads/%d.lz4", id)); err != nil {
					logger.Warn.Printf("unable to remove file: %v\n", err)
				}
			}
		}
	}()

	//check if guild has chat messages save turned on
	var isChatSaveOn bool
	if err := db.Db.QueryRow("SELECT save_chat FROM guilds WHERE id=$1", guildId).Scan(&isChatSaveOn); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	for _, file := range files {
		var attachment events.Attachment
		attachment.Filename = file.Filename
		attachment.Id = uid.Snowflake.Generate().Int64()
		msg.Attachments = append(msg.Attachments, attachment)
		//compress the file using LZ4 now

		fileContents, err := file.Open()
		if err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
		defer fileContents.Close()
		fileBytes, err := io.ReadAll(fileContents)
		if err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
		filesize := len(fileBytes) //possible bug file.Size but its a int64 review later
		compressedBuffer := make([]byte, lz4.CompressBlockBound(filesize))
		_, err = lz4.CompressBlock(fileBytes, compressedBuffer, nil)
		if err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}

		fileIds = append(fileIds, attachment.Id)

		outFile, err := os.Create(fmt.Sprintf("uploads/%d.lz4", attachment.Id))
		//TODO: delete files if failed or write them after when its deemed successful
		if err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
		defer outFile.Close()

		_, err = outFile.Write(compressedBuffer)
		if err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}

		//make files temporary if chat messages save turned off

		if _, err := tx.ExecContext(ctx, "INSERT INTO files (id, filename, created, temp, filesize) VALUES ($1, $2, $3, $4, $5)", attachment.Id, attachment.Filename, msg.Created, !isChatSaveOn, filesize); err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO msgfiles (file_id, msg_id) VALUES ($1, $2)", attachment.Id, msg.MsgId); err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
		msg.Attachments = append(msg.Attachments, attachment)
	}

	logger.Debug.Printf("Message recieved %s\n", msg.Content)
	if len(msg.Content) == 0 {
		logger.Error.Println(errors.ErrNoMsgContent)
		c.JSON(http.StatusUnprocessableEntity, errors.Body{
			Error:  errors.ErrNoMsgContent.Error(),
			Status: errors.StatusNoMsgContent,
		})
		return
	}

	if len(msg.Content) > config.Config.Guild.MaxMsgLength {
		logger.Error.Println(errors.ErrMsgTooLong)
		c.JSON(http.StatusForbidden, errors.Body{
			Error:  errors.ErrMsgTooLong.Error(),
			Status: errors.StatusMsgTooLong,
		})
		return
	}

	//finding mentions
	mentions := events.MentionExp.FindAllString(msg.Content, -1)
	mentionsEveryone := events.MentionEveryoneExp.MatchString(msg.Content)
	if len(mentions) > 0 {
		for _, mention := range mentions {
			mentionUserId, err := strconv.ParseInt(mention, 10, 64)
			if err != nil {
				logger.Error.Println(err)
				c.JSON(http.StatusBadRequest, errors.Body{
					Error:  err.Error(),
					Status: errors.StatusBadRequest,
				})
				return
			}
			var mentionUser events.User
			mentionUser.UserId = mentionUserId
			if err := db.Db.QueryRow("SELECT username FROM users WHERE id = $1", mentionUserId).Scan(&mentionUser.Name); err != nil && err != sql.ErrNoRows {
				logger.Error.Println(err)
				c.JSON(http.StatusInternalServerError, errors.Body{
					Error:  err.Error(),
					Status: errors.StatusInternalError,
				})
				return
			} else if err == sql.ErrNoRows {
				logger.Error.Println(errors.ErrUserNotFound)
				c.JSON(http.StatusBadRequest, errors.Body{
					Error:  errors.ErrUserNotFound.Error(),
					Status: errors.StatusBadRequest,
				})
				return
			}

			if _, err := tx.ExecContext(ctx, "INSERT IGNORE INTO msgmentions (msg_id, user_id) VALUES ($1, $2)", msg.MsgId, mentionUserId); err != nil {
				logger.Error.Println(err)
				c.JSON(http.StatusInternalServerError, errors.Body{
					Error:  err.Error(),
					Status: errors.StatusInternalError,
				})
				return
			}

			msg.Mentions = append(msg.Mentions, mentionUser)
		}
	}

	if isChatSaveOn {
		if _, err := tx.ExecContext(ctx, "INSERT INTO msgs (id, content, user_id, guild_id, created, mentions_everyone) VALUES ($1, $2, $3, $4, $5, $6)", msg.MsgId, msg.Content, user.Id, guildId, msg.Created, mentionsEveryone); err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
	}

	msg.MsgSaved = isChatSaveOn //false not saved | true saved

	var authorBody events.User
	var imageId sql.NullInt64
	if err := db.Db.QueryRow("SELECT username, image_id FROM users WHERE id=$1", user.Id).Scan(&authorBody.Name, &imageId); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	if imageId.Valid {
		authorBody.ImageId = imageId.Int64
	} else {
		authorBody.ImageId = -1
	}
	authorBody.UserId = user.Id
	msg.Author = authorBody

	if err := tx.Commit(); err != nil { //commits the transaction
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	fileSucessful = true
	wsclient.Pools.BroadcastGuild(intGuildId, wsclient.DataFrame{
		Op:    wsclient.TYPE_DISPATCH,
		Data:  msg,
		Event: events.CREATE_GUILD_MESSAGE,
	})
	c.Status(http.StatusNoContent)
}
