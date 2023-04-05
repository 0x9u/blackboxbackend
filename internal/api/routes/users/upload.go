package users

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/asianchinaboi/backendserver/internal/api/middleware"
	"github.com/asianchinaboi/backendserver/internal/db"
	"github.com/asianchinaboi/backendserver/internal/errors"
	"github.com/asianchinaboi/backendserver/internal/logger"
	"github.com/asianchinaboi/backendserver/internal/session"
	"github.com/asianchinaboi/backendserver/internal/uid"
	"github.com/gin-gonic/gin"
	"github.com/pierrec/lz4/v4"
)

func Upload(c *gin.Context) {
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
	file, err := c.FormFile("file")
	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusBadRequest,
			errors.Body{
				Error:  err.Error(),
				Status: errors.StatusBadRequest,
			})
		return
	}
	fileId := uid.Snowflake.Generate().Int64()
	created := time.Now().Unix()

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

	outFile, err := os.Create(fmt.Sprintf("uploads/%d.lz4", fileId))

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

	defer func() {
		if err := os.Remove(fmt.Sprintf("uploads/%d.lz4", fileId)); err != nil {
			logger.Warn.Printf("unable to remove file: %v\n", err)
		}
	}()

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

	if _, err := tx.ExecContext(ctx, "INSERT INTO files (id, filename, created, temp, filesize) VALUES ($1, $2, $3, $4, $5)", fileId, file.Filename, created, false, filesize); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	if _, err := tx.ExecContext(ctx, "UPDATE users SET image_id = $1 WHERE id = $2", fileId, user.Id); err != nil {
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
	c.Status(http.StatusNoContent)
}
