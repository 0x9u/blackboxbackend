package users

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
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
	"golang.org/x/crypto/bcrypt"
)

type editSelfBody struct {
	NewPassword *string `json:"newPassword"`
	Password    *string `json:"password"`
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
	var imageHeader *multipart.FileHeader

	contentType := c.GetHeader("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		var err error
		if imageHeader, err = c.FormFile("image"); err != nil && err != http.ErrMissingFile {
			logger.Error.Println(err)
			c.JSON(http.StatusBadRequest, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusBadRequest,
			})
			return
		}
		jsonData := c.PostForm("body")
		if err := json.Unmarshal([]byte(jsonData), &body); err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusBadRequest, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusBadRequest,
			})
			return
		}
	} else if strings.HasPrefix(contentType, "application/json") {
		if err := c.ShouldBindJSON(&body); err != nil {
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

	if body.NewPassword == nil && body.Email == nil && body.Username == nil && imageHeader == nil {
		logger.Error.Println(errors.ErrAllFieldsEmpty)
		c.JSON(http.StatusBadRequest, errors.Body{
			Error:  errors.ErrAllFieldsEmpty.Error(),
			Status: errors.StatusAllFieldsEmpty,
		})
		return
	}
	newUserInfo := events.User{}

	if body.Password != nil {
		var oldhashedpass string
		if err := db.Db.QueryRow("SELECT password FROM users WHERE id=$1", user.Id).Scan(&oldhashedpass); err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
		if err := bcrypt.CompareHashAndPassword([]byte(oldhashedpass), []byte(*body.Password)); err != nil {
			logger.Error.Println(errors.ErrInvalidDetails)
			c.JSON(http.StatusUnauthorized, errors.Body{
				Error:  errors.ErrInvalidDetails.Error(),
				Status: errors.StatusInvalidDetails,
			})
			return
		}
	} else {
		logger.Error.Println(errors.ErrInvalidDetails)
		c.JSON(http.StatusUnauthorized, errors.Body{
			Error:  errors.ErrInvalidDetails.Error(),
			Status: errors.StatusInvalidDetails,
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
	successful := false

	if body.NewPassword != nil {

		hashedPass, err := bcrypt.GenerateFromPassword([]byte(*body.NewPassword), bcrypt.DefaultCost)
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
		newUserInfo.Email = body.Email
	} else {
		if err := db.Db.QueryRow("SELECT email FROM users WHERE id=$1", user.Id).Scan(&newUserInfo.Email); err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
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
	} else {
		if err := db.Db.QueryRow("SELECT username FROM users WHERE id=$1", user.Id).Scan(&newUserInfo.Name); err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
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
	} else {
		var options int
		if err := db.Db.QueryRow("SELECT options FROM users WHERE id=$1", user.Id).Scan(&options); err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
		newUserInfo.Options = &options
	}

	if imageHeader != nil {
		//get old image id
		var oldImageId int64
		if err := db.Db.QueryRow("SELECT id FROM files WHERE user_id = $1", user.Id).Scan(&oldImageId); err != nil && err != sql.ErrNoRows {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		} else if err == sql.ErrNoRows {
			oldImageId = -1
		} else {
			defer func() { //defer just in case something went wrong
				if successful {
					deleteImageId := oldImageId
					if deleteImageId != -1 {
						if err := os.Remove(fmt.Sprintf("uploads/user/%d.lz4", deleteImageId)); err != nil {
							logger.Warn.Printf("failed to remove file: %v\n", err)
						}
					}
				}
			}()
			if _, err := tx.ExecContext(ctx, "DELETE FROM files WHERE id = $1", oldImageId); err != nil {
				logger.Error.Println(err)
				c.JSON(http.StatusInternalServerError, errors.Body{
					Error:  err.Error(),
					Status: errors.StatusInternalError,
				})
				return
			}
		}

		filename := imageHeader.Filename
		fileType := filepath.Ext(filename)

		imageId := uid.Snowflake.Generate().Int64()

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
		fileMIMEType := http.DetectContentType(fileBytes)

		if valid := files.ValidateImage(fileBytes, fileType); !valid {
			logger.Error.Println(errors.ErrFileInvalid)
			c.JSON(http.StatusBadRequest, errors.Body{
				Error:  errors.ErrFileInvalid.Error(),
				Status: errors.StatusFileInvalid,
			})
			return
		}

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

		if _, err = tx.ExecContext(ctx, "INSERT INTO files (id, filename, created, temp, filesize, user_id, filetype, entity_type) VALUES ($1, $2, now() , $3, $4, $5, $6,'user')", imageId, filename, false, filesize, user.Id, fileMIMEType); err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}

		outFile, err := os.Create(fmt.Sprintf("uploads/user/%d.lz4", imageId))
		if err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
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
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}

		newUserInfo.ImageId = imageId
	} else {
		var imageId int64
		if err := db.Db.QueryRow("SELECT id FROM files WHERE user_id=$1", user.Id).Scan(&imageId); err != nil && err != sql.ErrNoRows {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		} else if err == sql.ErrNoRows {
			newUserInfo.ImageId = -1
		} else {
			newUserInfo.ImageId = imageId
		}
	}

	if err := db.Db.QueryRow("SELECT flags FROM users WHERE id=$1", user.Id).Scan(&newUserInfo.Flags); err != nil {
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
	successful = true

	newUserInfo.UserId = user.Id

	res := wsclient.DataFrame{
		Op:    wsclient.TYPE_DISPATCH,
		Data:  newUserInfo,
		Event: events.UPDATE_SELF_USER_INFO,
	}

	wsclient.Pools.BroadcastClient(user.Id, res)

	newUserInfoOtherRes := newUserInfo

	newUserInfoOtherRes.Options = nil
	newUserInfoOtherRes.Email = nil

	otherRes := wsclient.DataFrame{
		Op:    wsclient.TYPE_DISPATCH,
		Data:  newUserInfoOtherRes,
		Event: events.UPDATE_USER_INFO,
	}

	userIdRows, err := db.Db.Query(
		`(SELECT DISTINCT userguilds.user_id AS user_id FROM userguilds WHERE EXISTS (SELECT 1 FROM userguilds AS ug2 WHERE ug2.user_id = $1 AND ug2.guild_id = userguilds.guild_id) AND userguilds.user_id != $1)
		UNION (SELECT DISTINCT friend_id AS user_id FROM friends WHERE user_id = $1)
		`, user.Id)
	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	defer userIdRows.Close()
	for userIdRows.Next() {
		var userId int64
		if err := userIdRows.Scan(&userId); err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
		wsclient.Pools.BroadcastClient(userId, otherRes)
	}
	c.Status(http.StatusNoContent)
}
