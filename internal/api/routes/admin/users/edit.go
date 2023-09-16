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
	"regexp"
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
	"golang.org/x/crypto/bcrypt"
)

type editUserBody struct {
	Password *string `json:"password"`
	Email    *string `json:"email"`
	Username *string `json:"username"`
	Flags    *int    `json:"flags"`
}

func Edit(c *gin.Context) {
	user := c.MustGet(middleware.User).(*session.Session)
	if user == nil {
		errors.SendErrorResponse(c, errors.ErrSessionDidntPass, errors.StatusInternalError)
		return
	}
	if !user.Perms.Admin && !user.Perms.Users.Edit {
		errors.SendErrorResponse(c, errors.ErrNotAuthorised, errors.StatusNotAuthorised)
		return
	}
	var body editUserBody
	var imageHeader *multipart.FileHeader

	contentType := c.GetHeader("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		var err error
		if imageHeader, err = c.FormFile("image"); err != nil && err != http.ErrMissingFile {
			errors.SendErrorResponse(c, err, errors.StatusBadRequest)
			return
		}
		jsonData := c.PostForm("body")
		if err := json.Unmarshal([]byte(jsonData), &body); err != nil {
			errors.SendErrorResponse(c, err, errors.StatusBadRequest)
			return
		}
	} else if strings.HasPrefix(contentType, "application/json") {
		if err := c.ShouldBindJSON(&body); err != nil {
			errors.SendErrorResponse(c, err, errors.StatusBadRequest)
			return
		}
	} else {
		errors.SendErrorResponse(c, errors.ErrNotSupportedContentType, errors.StatusBadRequest)
		return
	}
	userId := c.Param("userId")
	if match, err := regexp.MatchString("^[0-9]+$", userId); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	} else if !match {
		errors.SendErrorResponse(c, errors.ErrRouteParamInvalid, errors.StatusRouteParamInvalid)
		return
	}

	var userExists bool

	if err := db.Db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id=$1)", userId).Scan(&userExists); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}
	if !userExists {
		errors.SendErrorResponse(c, errors.ErrUserNotFound, errors.StatusUserNotFound)
		return
	}

	if body.Password == nil && body.Email == nil && body.Username == nil && body.Flags == nil && imageHeader == nil {
		errors.SendErrorResponse(c, errors.ErrAllFieldsEmpty, errors.StatusAllFieldsEmpty)
		return
	}
	newUserInfo := events.User{}

	//BEGIN TRANSACTION
	ctx := context.Background()
	tx, err := db.Db.BeginTx(ctx, nil)
	if err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}
	defer tx.Rollback() //rollback changes if failed
	successful := false

	if body.Password != nil {
		hashedPass, err := bcrypt.GenerateFromPassword([]byte(*body.Password), bcrypt.DefaultCost)
		if err != nil {
			errors.SendErrorResponse(c, err, errors.StatusInternalError)
			return
		}
		strHashedPass := string(hashedPass)

		if _, err = db.Db.Exec("UPDATE users SET password=$1 WHERE id=$2", strHashedPass, userId); err != nil {
			errors.SendErrorResponse(c, err, errors.StatusInternalError)
			return
		}
	}
	if body.Email != nil {
		var taken bool
		if err := db.Db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE email=$1)", *body.Email).Scan(&taken); err != nil {
			errors.SendErrorResponse(c, err, errors.StatusInternalError)
			return
		}
		if taken {
			errors.SendErrorResponse(c, errors.ErrEmailExists, errors.StatusEmailExists)
			return
		}
		if _, err = db.Db.Exec("UPDATE users SET email=$1 WHERE id=$2", *body.Email, userId); err != nil {
			errors.SendErrorResponse(c, err, errors.StatusInternalError)
			return
		}
		newUserInfo.Email = body.Email
	} else {
		if err := db.Db.QueryRow("SELECT email FROM users WHERE id=$1", userId).Scan(&newUserInfo.Email); err != nil {
			errors.SendErrorResponse(c, err, errors.StatusInternalError)
			return
		}
	}
	if body.Username != nil {
		var taken bool
		if err := db.Db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE username=$1)", *body.Username).Scan(&taken); err != nil {
			errors.SendErrorResponse(c, err, errors.StatusInternalError)
			return
		}
		if taken {
			errors.SendErrorResponse(c, errors.ErrUsernameExists, errors.StatusUsernameExists)
			return
		}
		if _, err = tx.ExecContext(ctx, "UPDATE users SET username=$1 WHERE id=$2", *body.Username, userId); err != nil {
			errors.SendErrorResponse(c, err, errors.StatusInternalError)
			return
		}
		newUserInfo.Name = *body.Username
	} else {
		if err := db.Db.QueryRow("SELECT username FROM users WHERE id=$1", userId).Scan(&newUserInfo.Name); err != nil {
			errors.SendErrorResponse(c, err, errors.StatusInternalError)
			return
		}
	}
	if body.Flags != nil {
		if _, err = tx.ExecContext(ctx, "UPDATE users SET flags=$1 WHERE id=$2", *body.Flags, userId); err != nil {
			errors.SendErrorResponse(c, err, errors.StatusInternalError)
			return
		}
		*newUserInfo.Flags = *body.Flags
	} else {
		if err := db.Db.QueryRow("SELECT flags FROM users WHERE id=$1", userId).Scan(&newUserInfo.Flags); err != nil {
			errors.SendErrorResponse(c, err, errors.StatusInternalError)
			return
		}
	}
	if imageHeader != nil {
		//get old image id
		var oldImageId int64
		if err := db.Db.QueryRow("SELECT id FROM files WHERE user_id = $1", user.Id).Scan(&oldImageId); err != nil && err != sql.ErrNoRows {
			errors.SendErrorResponse(c, err, errors.StatusInternalError)
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
				errors.SendErrorResponse(c, err, errors.StatusInternalError)
				return
			}
		}

		filename := imageHeader.Filename
		fileType := filepath.Ext(filename)
		imageCreated := time.Now().Unix()
		imageId := uid.Snowflake.Generate().Int64()

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

		if valid := files.ValidateImage(fileBytes, fileType); !valid {
			errors.SendErrorResponse(c, err, errors.StatusFileInvalid)
			return
		}

		filesize := len(fileBytes)

		if filesize > config.Config.Server.MaxFileSize {
			errors.SendErrorResponse(c, err, errors.StatusFileTooLarge)
			return
		} else if !(filesize >= 0) {
			errors.SendErrorResponse(c, err, errors.StatusFileNoBytes)
			return
		}

		compressedBuffer, err := files.Compress(fileBytes, filesize)
		if err != nil {
			errors.SendErrorResponse(c, err, errors.StatusInternalError)
			return
		}

		if _, err = tx.ExecContext(ctx, "INSERT INTO files (id, filename, created, temp, filesize, user_id, entity_type) VALUES ($1, $2, $3, $4, $5, $6,'user')", imageId, filename, imageCreated, false, filesize, user.Id); err != nil {
			errors.SendErrorResponse(c, err, errors.StatusInternalError)
			return
		}

		outFile, err := os.Create(fmt.Sprintf("uploads/user/%d.lz4", imageId))
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

		newUserInfo.ImageId = imageId
	} else {
		var imageId int64
		if err := db.Db.QueryRow("SELECT id FROM files WHERE user_id=$1", user.Id).Scan(&imageId); err != nil && err != sql.ErrNoRows {
			errors.SendErrorResponse(c, err, errors.StatusInternalError)
			return
		} else if err == sql.ErrNoRows {
			imageId = -1
		} else {
			newUserInfo.ImageId = imageId
		}
	}

	if err := tx.Commit(); err != nil { //commits the transaction
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
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
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}
	defer userIdRows.Close()
	for userIdRows.Next() {
		var userId int64
		if err := userIdRows.Scan(&userId); err != nil {
			errors.SendErrorResponse(c, err, errors.StatusInternalError)
			return
		}
		wsclient.Pools.BroadcastClient(userId, otherRes)
	}
	c.Status(http.StatusNoContent)
}
