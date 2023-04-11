package users

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
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
	"golang.org/x/crypto/bcrypt"
)

type editSelfBody struct {
	Password    *string `json:"password"`
	OldPassword *string `json:"oldPassword"`
	Email       *string `json:"email"`
	Username    *string `json:"username"`
	Options     *int    `json:"options"`
}

func editSelf(c *gin.Context) {
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
	var body editSelfBody

	if err := c.ShouldBindJSON(&body); err != nil {
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

	if body.Password == nil && body.Email == nil && body.Username == nil && imageHeader == nil {
		logger.Error.Println(errors.ErrAllFieldsEmpty)
		c.JSON(http.StatusBadRequest, errors.Body{
			Error:  errors.ErrAllFieldsEmpty.Error(),
			Status: errors.StatusAllFieldsEmpty,
		})
		return
	}
	newUserInfo := events.User{}

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
	successful := false
	if body.Password != nil && body.OldPassword != nil {
		var oldhashedpass string
		if err := db.Db.QueryRow("SELECT password FROM users WHERE id=$1", user.Id).Scan(&oldhashedpass); err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
		if err := bcrypt.CompareHashAndPassword([]byte(oldhashedpass), []byte(*body.OldPassword)); err != nil {
			logger.Error.Println(errors.ErrInvalidDetails)
			c.JSON(http.StatusUnauthorized, errors.Body{
				Error:  errors.ErrInvalidDetails.Error(),
				Status: errors.StatusInvalidDetails,
			})
			return
		}

		hashedPass, err := bcrypt.GenerateFromPassword([]byte(*body.Password), bcrypt.DefaultCost)
		if err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
		strHashedPass := string(hashedPass)

		if _, err = tx.ExecContext(ctx, "UPDATE users SET password=$1 WHERE id=$2", strHashedPass, user.Id); err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
	}
	if body.Email != nil {
		var taken bool
		if err := db.Db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE email=$1)", *body.Email).Scan(&taken); err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
		if taken {
			logger.Error.Println(errors.ErrEmailExists)
			c.JSON(http.StatusConflict, errors.Body{
				Error:  errors.ErrEmailExists.Error(),
				Status: errors.StatusEmailExists,
			})
			return
		}
		if _, err := tx.ExecContext(ctx, "UPDATE users SET email=$1 WHERE id=$2", *body.Email, user.Id); err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
		newUserInfo.Email = *body.Email
	}
	if body.Username != nil {
		var taken bool
		if err := db.Db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE username=$1)", *body.Username).Scan(&taken); err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
		if taken {
			logger.Error.Println(errors.ErrUsernameExists)
			c.JSON(http.StatusConflict, errors.Body{
				Error:  errors.ErrUsernameExists.Error(),
				Status: errors.StatusUsernameExists,
			})
			return
		}
		if _, err := tx.ExecContext(ctx, "UPDATE users SET username=$1 WHERE id=$2", *body.Username, user.Id); err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
		newUserInfo.Name = *body.Username
	}
	if body.Options != nil {
		if _, err := tx.ExecContext(ctx, "UPDATE users SET options=$1 WHERE id=$2", *body.Options, user.Id); err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
		newUserInfo.Options = body.Options
	}
	if imageHeader != nil {
		//get old image id
		var oldImageId sql.NullInt64
		if err := db.Db.QueryRow("SELECT image_id FROM users WHERE id = $1", user.Id).Scan(&oldImageId); err != nil {
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
		if _, err = tx.ExecContext(ctx, "UPDATE users SET image_id=$1 WHERE id=$2", imageId, user.Id); err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
		newUserInfo.ImageId = imageId
	} else {
		var imageId sql.NullInt64
		if err := db.Db.QueryRow("SELECT image_id FROM guilds WHERE id=$1", user.Id).Scan(&imageId); err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
		if imageId.Valid {
			newUserInfo.ImageId = imageId.Int64
		} else {
			newUserInfo.ImageId = -1
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
	res := wsclient.DataFrame{
		Op:    wsclient.TYPE_DISPATCH,
		Data:  newUserInfo,
		Event: events.UPDATE_USER_INFO,
	}
	wsclient.Pools.BroadcastClient(user.Id, res)
	c.Status(http.StatusNoContent)
}
