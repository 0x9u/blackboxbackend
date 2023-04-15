package files

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/asianchinaboi/backendserver/internal/db"
	"github.com/asianchinaboi/backendserver/internal/errors"
	"github.com/asianchinaboi/backendserver/internal/logger"
	"github.com/gin-gonic/gin"
	"github.com/pierrec/lz4/v4"
)

func get(c *gin.Context) {
	/*user := c.MustGet(middleware.User).(*session.Session)
	if user == nil {
		logger.Error.Println("user token not sent in data")
		c.JSON(http.StatusInternalServerError,
			errors.Body{
				Error:  errors.ErrSessionDidntPass.Error(),
				Status: errors.StatusInternalError,
			})
		return
	}*/

	fileId := c.Param("fileId")
	if match, err := regexp.MatchString("^[0-9]+$", fileId); err != nil {
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
	entityType := c.Param("entityType")
	if match, err := regexp.MatchString("^(guild|user|msg)$", entityType); err != nil {
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
	var filename string
	var filesize int64
	if err := db.Db.QueryRow("SELECT filename, filesize FROM files WHERE id = $1 AND entity_type = $2", fileId, entityType).Scan(&filename, &filesize); err != nil && err != sql.ErrNoRows {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	} else if err == sql.ErrNoRows {
		logger.Error.Println(errors.ErrFileNotFound)
		c.JSON(http.StatusNotFound, errors.Body{
			Error:  errors.ErrFileNotFound.Error(),
			Status: errors.StatusFileNotFound,
		})
		return
	}
	/*
		file, err := os.Open(fmt.Sprintf("uploads/%s/%s.lz4", entityType, fileId))
		if err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
		defer file.Close()
		var uncompressedBuffer bytes.Buffer
		uncompressor := lz4.NewReader(file)
		if _, err := uncompressor.WriteTo(&uncompressedBuffer); err != nil {
			logger.Error.Println(err)
			logger.Debug.Println("uncompressedBuffer:", uncompressedBuffer)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
		//might be a memory leak no close apparently for NewReader
	*/
	fileBytes, err := os.ReadFile(fmt.Sprintf("uploads/%s/%s.lz4", entityType, fileId))
	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	uncompressedBuffer := make([]byte, filesize)

	if _, err := lz4.UncompressBlock(fileBytes, uncompressedBuffer); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	var contentType string

	if strings.HasSuffix(filename, ".png") {
		contentType = "image/png"
	} else if strings.HasSuffix(filename, ".jpg") || strings.HasSuffix(filename, ".jpeg") {
		contentType = "image/jpeg"
	} else if strings.HasSuffix(filename, ".gif") {
		contentType = "image/gif"
	} else if strings.HasSuffix(filename, ".mp4") {
		contentType = "video/mp4"
	} else if strings.HasSuffix(filename, ".webm") {
		contentType = "video/webm"
	} else if strings.HasSuffix(filename, ".ogg") {
		contentType = "video/ogg"
	} else if strings.HasSuffix(filename, ".mp3") {
		contentType = "audio/mpeg"
	} else if strings.HasSuffix(filename, ".wav") {
		contentType = "audio/wav"
	} else if strings.HasSuffix(filename, ".ogg") {
		contentType = "audio/ogg"
	} else if strings.HasSuffix(filename, ".pdf") {
		contentType = "application/pdf"
	} else if strings.HasSuffix(filename, ".txt") {
		contentType = "text/plain"
	} else {
		contentType = "application/octet-stream"
	}

	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	c.Data(http.StatusOK, "application/octet-stream", uncompressedBuffer)
}
