package users

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

	"github.com/asianchinaboi/backendserver/internal/config"
	"github.com/asianchinaboi/backendserver/internal/db"
	"github.com/asianchinaboi/backendserver/internal/errors"
	"github.com/asianchinaboi/backendserver/internal/events"
	"github.com/asianchinaboi/backendserver/internal/files"
	"github.com/asianchinaboi/backendserver/internal/logger"
	"github.com/asianchinaboi/backendserver/internal/session"
	"github.com/asianchinaboi/backendserver/internal/uid"
	"github.com/gin-gonic/gin"
)

func userCreate(c *gin.Context) {
	logger.Debug.Println("Creating User")
	var user events.User
	var imageHeader *multipart.FileHeader

	contentType := c.GetHeader("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		var err error
		if imageHeader, err = c.FormFile("image"); err != nil && err != http.ErrMissingFile {
			errors.SendErrorResponse(c, err, errors.StatusBadRequest)
			return
		}
		jsonData := c.PostForm("body")
		if err := json.Unmarshal([]byte(jsonData), &user); err != nil {
			errors.SendErrorResponse(c, err, errors.StatusBadRequest)
			return
		}
	} else if strings.HasPrefix(contentType, "application/json") {
		if err := c.ShouldBindJSON(&user); err != nil {
			errors.SendErrorResponse(c, err, errors.StatusBadRequest)
			return
		}
	} else {
		errors.SendErrorResponse(c, errors.ErrNotSupportedContentType, errors.StatusBadRequest)
		return
	}

	successful := false

	if user.Email == nil {
		user.Email = new(string)
	}

	//validation

	if statusCode, err := events.ValidateUserInput(user); err != nil {
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

	var isUsernameTaken bool

	if err := db.Db.QueryRow("SELECT EXISTS (SELECT 1 FROM users WHERE username=$1)", user.Name).Scan(&isUsernameTaken); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	if isUsernameTaken {
		errors.SendErrorResponse(c, errors.ErrUsernameExists, errors.StatusUsernameExists)
		return
	}

	hashedPass := hashPass(user.Password)

	user.UserId = uid.Snowflake.Generate().Int64()

	if _, err := tx.ExecContext(ctx, "INSERT INTO users (id, email, password, username) VALUES ($1, $2, $3, $4)", user.UserId, *user.Email, hashedPass, user.Name); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	if imageHeader != nil {
		imageId := uid.Snowflake.Generate().Int64()

		filename := imageHeader.Filename
		fileType := filepath.Ext(filename)

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

		outFile, err := os.Create(fmt.Sprintf("uploads/user/%d.lz4", imageId))
		//TODO: delete files if failed or write them after when its deemed successful
		if err != nil {
			errors.SendErrorResponse(c, err, errors.StatusInternalError)
			return
		}
		defer func() { //defer just in case something went wrong
			if !successful {
				if err := os.Remove(fmt.Sprintf("uploads/user/%d.lz4", imageId)); err != nil {
					logger.Warn.Printf("failed to remove file: %v\n", err)
				}
			}
		}()
		defer outFile.Close()

		if _, err = outFile.Write(compressedBuffer); err != nil {
			errors.SendErrorResponse(c, err, errors.StatusInternalError)
			return
		}

		if _, err = tx.ExecContext(ctx, "INSERT INTO files (id, user_id, filename, created, temp, filesize, filetype, entity_type) VALUES ($1, $2, $3, NOW() , $4, $5, $6, 'user')", imageId, user.UserId, filename, false, filesize, fileMIMEType); err != nil {
			errors.SendErrorResponse(c, err, errors.StatusInternalError)
			return
		}
	}

	//add user role

	if _, err := tx.ExecContext(ctx, "INSERT INTO userroles (user_id, role_id) VALUES ($1, 1)", user.UserId); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	if err := tx.Commit(); err != nil { //commits the transaction
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	successful = true

	//create session for new user
	authData, err := session.GenToken(user.UserId)
	if err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	c.JSON(http.StatusOK, authData)
}
