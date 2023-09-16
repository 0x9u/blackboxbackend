package files

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"regexp"

	"github.com/asianchinaboi/backendserver/internal/db"
	"github.com/asianchinaboi/backendserver/internal/errors"
	"github.com/gin-gonic/gin"
	"github.com/pierrec/lz4/v4"
)

func get(c *gin.Context) {

	fileId := c.Param("fileId")
	if match, err := regexp.MatchString("^[0-9]+$", fileId); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	} else if !match {
		errors.SendErrorResponse(c, errors.ErrRouteParamInvalid, errors.StatusRouteParamInvalid)
		return
	}
	entityType := c.Param("entityType")
	if match, err := regexp.MatchString("^(guild|user|msg)$", entityType); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	} else if !match {
		errors.SendErrorResponse(c, errors.ErrRouteParamInvalid, errors.StatusRouteParamInvalid)
		return
	}
	var filename string
	var filesize int64
	if err := db.Db.QueryRow("SELECT filename, filesize FROM files WHERE id = $1 AND entity_type = $2", fileId, entityType).Scan(&filename, &filesize); err != nil && err != sql.ErrNoRows {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	} else if err == sql.ErrNoRows {
		errors.SendErrorResponse(c, errors.ErrFileNotFound, errors.StatusFileNotFound)
		return
	}
	fileBytes, err := os.ReadFile(fmt.Sprintf("uploads/%s/%s.lz4", entityType, fileId))
	if err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	uncompressedBuffer := make([]byte, filesize)

	if _, err := lz4.UncompressBlock(fileBytes, uncompressedBuffer); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	contentType := http.DetectContentType(uncompressedBuffer)

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	c.Data(http.StatusOK, contentType, uncompressedBuffer)
}
