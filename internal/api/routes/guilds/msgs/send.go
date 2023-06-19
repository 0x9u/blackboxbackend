package msgs

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/asianchinaboi/backendserver/internal/api/middleware"
	"github.com/asianchinaboi/backendserver/internal/config"
	"github.com/asianchinaboi/backendserver/internal/db"
	"github.com/asianchinaboi/backendserver/internal/errors"
	"github.com/asianchinaboi/backendserver/internal/events"
	"github.com/asianchinaboi/backendserver/internal/files"
	"github.com/asianchinaboi/backendserver/internal/logger"
	"github.com/asianchinaboi/backendserver/internal/session"
	"github.com/asianchinaboi/backendserver/internal/uid"
	"github.com/asianchinaboi/backendserver/internal/wsclient"
	"github.com/gin-gonic/gin"
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
	var attachmentFiles []*multipart.FileHeader
	fileIds := []int64{}
	fileSucessful := false

	contentType := c.GetHeader("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		form, err := c.MultipartForm()
		if err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusBadRequest, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusBadRequest,
			})
			return
		}

		attachmentFiles = form.File["file"]
		msg.Attachments = []events.Attachment{}
		defer func() {
			if !fileSucessful {
				for _, id := range fileIds {
					if err := os.Remove(fmt.Sprintf("uploads/msg/%d.lz4", id)); err != nil {
						logger.Warn.Printf("unable to remove file: %v\n", err)
					}
				}
			}
		}()
		jsonData := c.PostForm("body")
		if err := json.Unmarshal([]byte(jsonData), &msg); err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusBadRequest, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusBadRequest,
			})
			return
		}
	} else if strings.HasPrefix(contentType, "application/json") {
		if err := c.ShouldBindJSON(&msg); err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusBadRequest, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusBadRequest,
			})
			return
		}
	} else {
		logger.Error.Println(errors.ErrNotSupportedContentType)
		c.JSON(http.StatusBadRequest, errors.Body{
			Error:  errors.ErrNotSupportedContentType.Error(),
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
	defer tx.Rollback() //rollback changes if failed

	msg.Content = strings.TrimSpace(msg.Content)
	//screw off html

	//msg.Content = html.EscapeString(msg.Content) //prevents xss attacks //not needed we are using react
	msg.MsgId = uid.Snowflake.Generate().Int64()
	//check if attachments uploaded

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

	if len(msg.Content) == 0 && len(attachmentFiles) == 0 {
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
	mentions := events.MentionExp.FindAllStringSubmatch(msg.Content, -1)
	logger.Debug.Println("msgcontent:", msg.Content)
	logger.Debug.Println("mentions:", mentions)
	msg.MentionsEveryone = events.MentionEveryoneExp.MatchString(msg.Content)

	if isChatSaveOn {
		if err := tx.QueryRowContext(ctx, "INSERT INTO msgs (id, content, user_id, guild_id, mentions_everyone) VALUES ($1, $2, $3, $4, $5) RETURNING created", msg.MsgId, msg.Content, user.Id, guildId, msg.MentionsEveryone).Scan(&msg.Created); err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
	}

	msg.Mentions = make([]events.User, 0, len(mentions))

	if len(mentions) > 0 {
		logger.Debug.Println("mentions found")
		seen := make(map[int64]bool)
		for _, mention := range mentions {
			mentionUserId, err := strconv.ParseInt(mention[1], 10, 64)
			if err != nil {
				logger.Error.Println(err)
				c.JSON(http.StatusInternalServerError, errors.Body{
					Error:  err.Error(),
					Status: errors.StatusInternalError,
				})
				return
			}
			if seen[mentionUserId] {
				continue
			}
			seen[mentionUserId] = true

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

			if isChatSaveOn {
				if _, err := tx.ExecContext(ctx, "INSERT INTO msgmentions (msg_id, user_id) VALUES ($1, $2)", msg.MsgId, mentionUserId); err != nil {
					logger.Error.Println(err)
					c.JSON(http.StatusInternalServerError, errors.Body{
						Error:  err.Error(),
						Status: errors.StatusInternalError,
					})
					return
				}
			}

			msg.Mentions = append(msg.Mentions, mentionUser)
		}
	}

	for _, file := range attachmentFiles {
		var attachment events.Attachment
		attachment.Filename = file.Filename
		attachment.Id = uid.Snowflake.Generate().Int64()

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

		attachment.Type = http.DetectContentType(fileBytes)
		logger.Debug.Println("uploaded type", attachment.Type)
		msg.Attachments = append(msg.Attachments, attachment)

		filesize := len(fileBytes)

		if filesize > config.Config.Server.MaxFileSize {
			logger.Error.Println(errors.ErrFileTooLarge)
			c.JSON(http.StatusRequestEntityTooLarge, errors.Body{
				Error:  errors.ErrFileTooLarge.Error(),
				Status: errors.StatusFileTooLarge,
			})
			return
		} else if !(filesize >= 0) {
			logger.Error.Println(errors.ErrFileNoBytes)
			c.JSON(http.StatusBadRequest, errors.Body{
				Error:  errors.ErrFileNoBytes.Error(),
				Status: errors.StatusFileNoBytes,
			})
			return
		}

		compressedBuffer, err := files.Compress(fileBytes, filesize)
		if err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}

		outFile, err := os.Create(fmt.Sprintf("uploads/msg/%d.lz4", attachment.Id))
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

		if isChatSaveOn {
			if _, err := tx.ExecContext(ctx, "INSERT INTO files (id, msg_id, filename, created, temp, filesize, filetype, entity_type) VALUES ($1, $2, $3, $4, $5, $6, $7, 'msg')", attachment.Id, msg.MsgId, attachment.Filename, msg.Created, !isChatSaveOn, filesize, attachment.Type); err != nil {
				logger.Error.Println(err)
				c.JSON(http.StatusInternalServerError, errors.Body{
					Error:  err.Error(),
					Status: errors.StatusInternalError,
				})
				return
			}
		} else {
			if _, err := tx.ExecContext(ctx, "INSERT INTO files (id, filename, created, temp, filesize, filetype ,entity_type) VALUES ($1, $2, $3, $4, $5, $6, 'msg')", attachment.Id, attachment.Filename, msg.Created, !isChatSaveOn, filesize, attachment.Type); err != nil {
				logger.Error.Println(err)
				c.JSON(http.StatusInternalServerError, errors.Body{
					Error:  err.Error(),
					Status: errors.StatusInternalError,
				})
				return
			}
		}

		fileIds = append(fileIds, attachment.Id)
	}

	var isDm bool

	if err := db.Db.QueryRow("SELECT receiver_id IS NOT NULL FROM userguilds WHERE guild_id = $1 AND user_id = $2", guildId, user.Id).Scan(&isDm); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	if isDm {
		if _, err := tx.ExecContext(ctx, "UPDATE userguilds SET left_dm = false WHERE guild_id = $1", guildId); err != nil {
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
	if err := db.Db.QueryRow("SELECT username, files.id FROM users LEFT JOIN files ON files.user_id = users.id WHERE users.id=$1", user.Id).Scan(&authorBody.Name, &imageId); err != nil {
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
	msg.GuildId = intGuildId

	if err := tx.Commit(); err != nil { //commits the transaction
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	fileSucessful = true

	var statusMessage string
	if isDm {
		statusMessage = events.CREATE_DM_MESSAGE
	} else {
		statusMessage = events.CREATE_GUILD_MESSAGE
	}

	wsclient.Pools.BroadcastGuild(intGuildId, wsclient.DataFrame{
		Op:    wsclient.TYPE_DISPATCH,
		Data:  msg,
		Event: statusMessage,
	})
	c.Status(http.StatusNoContent)
}
