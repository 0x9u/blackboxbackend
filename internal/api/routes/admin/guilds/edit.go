package guilds

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/asianchinaboi/backendserver/internal/api/middleware"
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

type editGuildBody struct {
	SaveChat *bool   `json:"saveChat"`
	Name     *string `json:"name"`
	OwnerId  *int64  `json:"ownerId"`
}

func Edit(c *gin.Context) {
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
	if !user.Perms.Admin && !user.Perms.Guilds.Edit {
		logger.Error.Println(errors.ErrNotAuthorised)
		c.JSON(http.StatusForbidden, errors.Body{
			Error:  errors.ErrNotAuthorised.Error(),
			Status: errors.StatusNotAuthorised,
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

	var newSettings editGuildBody
	if err := c.ShouldBindJSON(&newSettings); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusBadRequest, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusBadRequest,
		})
		return
	}

	imageHeader, err := c.FormFile("image")
	if err != nil && err != http.ErrMissingFile {
		logger.Error.Println(err)
		c.JSON(http.StatusBadRequest, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusBadRequest,
		})
		return
	}

	if newSettings.SaveChat == nil && newSettings.Name == nil && imageHeader == nil {
		logger.Error.Println(errors.ErrAllFieldsEmpty)
		c.JSON(http.StatusBadRequest, errors.Body{
			Error:  errors.ErrAllFieldsEmpty.Error(),
			Status: errors.StatusAllFieldsEmpty,
		})
		return
	}

	successful := false

	var exists bool
	if err := db.Db.QueryRow("SELECT EXISTS(SELECT 1 FROM guilds WHERE id = $1)", guildId).Scan(&exists); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	if !exists {
		logger.Error.Println(errors.ErrGuildNotExist)
		c.JSON(http.StatusNotFound, errors.Body{
			Error:  errors.ErrGuildNotExist.Error(),
			Status: errors.StatusGuildNotExist,
		})
		return
	}

	bodyRes := events.Guild{}
	intGuildId, err := strconv.ParseInt(guildId, 10, 64)
	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
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

	bodyRes.GuildId = intGuildId

	bodySettingsRes := events.Guild{}

	bodySettingsRes.GuildId = intGuildId

	if newSettings.SaveChat != nil {

		if _, err = tx.ExecContext(ctx, "UPDATE guilds SET save_chat=$1 WHERE id=$2", *newSettings.SaveChat, guildId); err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
		*bodySettingsRes.SaveChat = *newSettings.SaveChat
	} else {
		var saveChat bool
		if err := db.Db.QueryRow("SELECT save_chat FROM guilds WHERE id = $1", guildId).Scan(&saveChat); err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
		bodySettingsRes.SaveChat = &saveChat
	}
	if newSettings.Name != nil {
		if _, err = tx.ExecContext(ctx, "UPDATE guilds SET name=$1 WHERE id=$2", *newSettings.Name, guildId); err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
		bodyRes.Name = *newSettings.Name
	} else {
		var name string
		if err := db.Db.QueryRow("SELECT name FROM guilds WHERE id = $1", guildId).Scan(&name); err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
		bodyRes.Name = name
	}

	if imageHeader != nil {
		//get old image id
		var oldImageId sql.NullInt64
		if err := db.Db.QueryRow("SELECT image_id FROM guilds WHERE id = $1", guildId).Scan(&oldImageId); err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
		if oldImageId.Valid {
			defer func() { //defer just in case something went wrong
				if successful {
					deleteImageId := oldImageId.Int64
					if err := os.Remove(fmt.Sprintf("uploads/%d.lz4", deleteImageId)); err != nil {
						logger.Error.Println(err)
						c.JSON(http.StatusInternalServerError, errors.Body{
							Error:  err.Error(),
							Status: errors.StatusInternalError,
						})
						return
					}
				}
			}()

			if _, err := tx.ExecContext(ctx, "DELETE FROM files WHERE id = $1", oldImageId.Int64); err != nil {
				logger.Error.Println(err)
				c.JSON(http.StatusInternalServerError, errors.Body{
					Error:  err.Error(),
					Status: errors.StatusInternalError,
				})
				return
			}
		}
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

		imageId := uid.Snowflake.Generate().Int64()
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
		if _, err = tx.ExecContext(ctx, "UPDATE guilds SET image_id=$1 WHERE id=$2", imageId, guildId); err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
		bodyRes.ImageId = imageId
	} else {
		var imageId sql.NullInt64
		if err := db.Db.QueryRow("SELECT icon FROM guilds WHERE id=$1", guildId).Scan(&imageId); err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
		if imageId.Valid {
			bodyRes.ImageId = imageId.Int64
		} else {
			bodyRes.ImageId = -1
		}
	}

	if err := tx.Commit(); err != nil { //commits the transaction
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	successful = true

	guildRes := wsclient.DataFrame{
		Op:    wsclient.TYPE_DISPATCH,
		Data:  bodyRes,
		Event: events.UPDATE_GUILD,
	}
	wsclient.Pools.BroadcastGuild(intGuildId, guildRes)

	c.Status(http.StatusNoContent)
}
