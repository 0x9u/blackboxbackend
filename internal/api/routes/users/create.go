package users

import (
	"context"
	"database/sql"
	"io"
	"net/http"
	"time"

	"github.com/asianchinaboi/backendserver/internal/db"
	"github.com/asianchinaboi/backendserver/internal/errors"
	"github.com/asianchinaboi/backendserver/internal/events"
	"github.com/asianchinaboi/backendserver/internal/logger"
	"github.com/asianchinaboi/backendserver/internal/session"
	"github.com/asianchinaboi/backendserver/internal/uid"
	"github.com/gin-gonic/gin"
	"github.com/pierrec/lz4/v4"
)

func userCreate(c *gin.Context) {
	logger.Debug.Println("Creating User")
	var user events.User
	if err := c.ShouldBindJSON(&user); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusBadRequest, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusBadRequest,
		})
		return
	}

	var imageId sql.NullInt64
	imageHeader, err := c.FormFile("image")
	if err != nil && err != http.ErrMissingFile {
		logger.Error.Println(err)
		c.JSON(http.StatusBadRequest, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusBadRequest,
		})
		return
	}

	//validation

	if statusCode, err := events.ValidateUserInput(user); err != nil && statusCode != errors.StatusInternalError {
		logger.Error.Println(err)
		c.JSON(http.StatusUnprocessableEntity, errors.Body{
			Error:  err.Error(),
			Status: statusCode,
		})
		return
	} else if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	var isUsernameTaken bool

	if err := db.Db.QueryRow("SELECT EXISTS (SELECT 1 FROM users WHERE username=$1)", user.Name).Scan(&isUsernameTaken); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	if isUsernameTaken {
		logger.Error.Println(errors.ErrUsernameExists)
		c.JSON(http.StatusConflict, errors.Body{
			Error:  errors.ErrUsernameExists.Error(),
			Status: errors.StatusUsernameExists,
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
		if err != nil {
			if err := tx.Rollback(); err != nil {
				logger.Warn.Printf("unable to rollback error: %v\n", err)
			}
		}
	}() //rollback changes if failed

	hashedPass := hashPass(user.Password)

	user.UserId = uid.Snowflake.Generate().Int64()

	if imageHeader != nil {
		imageId.Int64 = uid.Snowflake.Generate().Int64()
		imageId.Valid = true

		filename := imageHeader.Filename
		imageCreated := time.Now().Unix()

		image, err := imageHeader.Open()
		if err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusBadRequest, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusBadRequest,
			})
			return
		}
		defer image.Close()

		fileBytes, err := io.ReadAll(image)
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

		if _, err = tx.ExecContext(ctx, "INSERT INTO files (id, filename, created, temp, filesize) VALUES ($1, $2, $3, $4, $5)", imageId, filename, imageCreated, false, filesize); err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
	} else {
		imageId.Int64 = -1
		imageId.Valid = false
	}

	if _, err := tx.ExecContext(ctx, "INSERT INTO users (id, email, password, username, image_id) VALUES ($1, $2, $3, $4)", user.UserId, user.Email, hashedPass, user.Name, imageId); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	//add user role

	if _, err := tx.ExecContext(ctx, "INSERT INTO userroles (user_id, role_id) VALUES ($1, 1)", user.UserId); err != nil {
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

	//create session for new user
	authData, err := session.GenToken(user.UserId)
	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	logger.Debug.Printf("info about new user %v\n", authData)
	c.JSON(http.StatusOK, authData)
}
