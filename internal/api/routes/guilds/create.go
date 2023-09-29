package guilds

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

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

//accepts name, icon, savechat

func createGuild(c *gin.Context) {
	user := c.MustGet(middleware.User).(*session.Session)
	if user == nil {
		errors.SendErrorResponse(c, errors.ErrSessionDidntPass, errors.StatusInternalError)
		return
	}

	var guild events.Guild
	var imageHeader *multipart.FileHeader

	contentType := c.GetHeader("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		var err error
		if imageHeader, err = c.FormFile("image"); err != nil && err != http.ErrMissingFile {
			errors.SendErrorResponse(c, err, errors.StatusBadRequest)
			return
		}
		jsonData := c.PostForm("body")
		if err := json.Unmarshal([]byte(jsonData), &guild); err != nil {
			errors.SendErrorResponse(c, err, errors.StatusBadRequest)
			return
		}
	} else if strings.HasPrefix(contentType, "application/json") {
		if err := c.ShouldBindJSON(&guild); err != nil {
			errors.SendErrorResponse(c, err, errors.StatusBadRequest)
			return
		}
	} else {
		errors.SendErrorResponse(c, errors.ErrNotSupportedContentType, errors.StatusBadRequest)
		return
	}

	if statusCode, err := events.ValidateGuildInput(&guild); err != nil {
		errors.SendErrorResponse(c, err, statusCode)
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

	successful := false

	guildId := uid.Snowflake.Generate().Int64()

	if _, err := tx.ExecContext(ctx, "INSERT INTO guilds (id, name, save_chat) VALUES ($1, $2, $3)", guildId, guild.Name, guild.SaveChat); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	if imageHeader != nil {
		imageId := uid.Snowflake.Generate().Int64()
		filename := imageHeader.Filename
		fileType := filepath.Ext(filename)
		imageCreated := time.Now().Unix()
		image, err := imageHeader.Open()
		if err != nil {
			errors.SendErrorResponse(c, err, errors.StatusBadRequest)
			return
		}
		defer image.Close()

		fileBytes, err := io.ReadAll(image)
		if err != nil {
			errors.SendErrorResponse(c, err, errors.StatusInternalError)
			return
		}

		fileMIMEType := http.DetectContentType(fileBytes)

		if valid := files.ValidateImage(fileBytes, fileType); !valid {
			errors.SendErrorResponse(c, errors.ErrFileInvalid, errors.StatusFileInvalid)
			return
		}

		filesize := len(fileBytes)

		if filesize > config.Config.Server.MaxFileSize {
			errors.SendErrorResponse(c, errors.ErrFileTooLarge, errors.StatusFileTooLarge)
			return
		} else if !(filesize >= 0) {
			errors.SendErrorResponse(c, errors.ErrFileNoBytes, errors.StatusFileNoBytes)
			return
		}

		compressedBuffer, err := files.Compress(fileBytes, filesize)
		if err != nil {
			errors.SendErrorResponse(c, err, errors.StatusInternalError)
			return
		}

		outFile, err := os.Create(fmt.Sprintf("uploads/guild/%d.lz4", imageId))
		//TODO: delete files if failed or write them after when its deemed successful
		if err != nil {
			errors.SendErrorResponse(c, err, errors.StatusInternalError)
			return
		}
		defer func() { //defer just in case something went wrong
			if !successful {
				if err := os.Remove(fmt.Sprintf("uploads/guild/%d.lz4", imageId)); err != nil {
					logger.Warn.Printf("failed to remove file: %v\n", err)
				}
			}
		}()
		defer outFile.Close()

		if _, err = outFile.Write(compressedBuffer); err != nil {
			errors.SendErrorResponse(c, err, errors.StatusInternalError)
			return
		}

		if _, err = tx.ExecContext(ctx, "INSERT INTO files (id, guild_id, filename, created, temp, filesize, filetype, entity_type) VALUES ($1, $2, $3, $4, $5, $6, $7, 'guild')", imageId, guildId, filename, imageCreated, false, filesize, fileMIMEType); err != nil {
			errors.SendErrorResponse(c, err, errors.StatusInternalError)
			return
		}
		guild.ImageId = imageId
	} else {
		logger.Debug.Println("no image provided")
		guild.ImageId = -1
	}

	invite := events.Invite{
		Invite:  session.GenerateRandString(10),
		GuildId: guildId,
	}

	if _, err := tx.ExecContext(ctx, "INSERT INTO unreadmsgs (guild_id, user_id) VALUES ($1, $2)", guildId, user.Id); err != nil { //cleanup if failed later
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO userguilds (guild_id, user_id, owner) VALUES ($1, $2, true)", guildId, user.Id); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO invites (invite, guild_id) VALUES ($1, $2)", invite.Invite, guildId); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	if err := tx.Commit(); err != nil { //commits the transaction
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	successful = true

	res := wsclient.DataFrame{
		Op: wsclient.TYPE_DISPATCH,
		Data: events.Guild{
			GuildId: guildId,
			OwnerId: user.Id,
			Name:    guild.Name,
			ImageId: guild.ImageId,
		},
		Event: events.GUILD_CREATE,
	}
	invRes := wsclient.DataFrame{
		Op:    wsclient.TYPE_DISPATCH,
		Data:  invite,
		Event: events.INVITE_CREATE,
	}
	wsclient.Pools.BroadcastClient(user.Id, res)
	//shit i forgot to create a pool
	wsclient.Pools.AddUserToGuildPool(guildId, user.Id)
	wsclient.Pools.BroadcastGuild(guildId, invRes)
	//possible race condition but shouldnt be possible since sql does it by queue
	c.Status(http.StatusNoContent) //writing this code at nearly 12 am gotta keep the grind up
	//dec 9 2022 writing code at nearly 12 am is not good im fixing it rn and holy crap some of the stuff looks shit
}
