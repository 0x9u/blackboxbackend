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

	dmId := c.Param("dmId")
	if match, err := regexp.MatchString("^[0-9]+$", dmId); err != nil {
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

	intDmId, err := strconv.ParseInt(dmId, 10, 64)
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
	logger.Debug.Printf("Message recieved %s\n", msg.Content)
	if len(msg.Content) == 0 {
		logger.Error.Println(errors.ErrNoMsgContent)
		c.JSON(http.StatusUnprocessableEntity, errors.Body{
			Error:  errors.ErrNoMsgContent.Error(),
			Status: errors.StatusNoMsgContent,
		})
		return
	}

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
		filesize := len(fileBytes)
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

		if _, err := tx.ExecContext(ctx, "INSERT INTO files (id, filename, created, temp, filesize) VALUES ($1, $2, $3, $4, $5)", attachment.Id, attachment.Filename, msg.Created, false, filesize); err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
		msg.Attachments = append(msg.Attachments, attachment)
	}

	if len(msg.Content) > config.Config.Guild.MaxMsgLength {
		logger.Error.Println(errors.ErrMsgTooLong)
		c.JSON(http.StatusForbidden, errors.Body{
			Error:  errors.ErrMsgTooLong.Error(),
			Status: errors.StatusMsgTooLong,
		})
		return
	}

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

			if _, err := tx.ExecContext(ctx, "INSERT IGNORE INTO directmsgmentions (directmsg_id, user_id) VALUES ($1, $2)", msg.MsgId, mentionUserId); err != nil {
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

	if _, err := tx.ExecContext(ctx, "INSERT INTO directmsgs (id, content, sender_id, dm_id, created, mentions_everyone) VALUES ($1, $2, $3, $4, $5)", msg.MsgId, msg.Content, user.Id, dmId, msg.Created, mentionsEveryone); err != nil {
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

	var authorBody events.User
	if err := db.Db.QueryRow("SELECT username, image_id FROM users WHERE id=$1", user.Id).Scan(&authorBody.Name, &authorBody.ImageId); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	authorBody.UserId = user.Id
	msg.Author = authorBody
	msg.DmId = intDmId

	var otherUser int64

	if err := db.Db.QueryRow("SELECT user_id FROM userdirectmsgsguild WHERE dm_id = $1 AND user_id != $2 ", dmId, user.Id).Scan(&otherUser); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	wsclient.Pools.BroadcastClient(user.Id, wsclient.DataFrame{
		Op:    wsclient.TYPE_DISPATCH,
		Data:  msg,
		Event: events.CREATE_DM_MESSAGE,
	})
	wsclient.Pools.BroadcastClient(otherUser, wsclient.DataFrame{
		Op:    wsclient.TYPE_DISPATCH,
		Data:  msg,
		Event: events.CREATE_DM_MESSAGE,
	})
	c.Status(http.StatusNoContent)
}
