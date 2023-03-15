package users

import (
	"net/http"
	"regexp"
	"strconv"

	"github.com/asianchinaboi/backendserver/internal/api/middleware"
	"github.com/asianchinaboi/backendserver/internal/db"
	"github.com/asianchinaboi/backendserver/internal/errors"
	"github.com/asianchinaboi/backendserver/internal/events"
	"github.com/asianchinaboi/backendserver/internal/logger"
	"github.com/asianchinaboi/backendserver/internal/session"
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
		logger.Error.Println("user token not sent in data")
		c.JSON(http.StatusInternalServerError,
			errors.Body{
				Error:  errors.ErrSessionDidntPass.Error(),
				Status: errors.StatusInternalError,
			})
		return
	}
	if !user.Perms.Admin && !user.Perms.Users.Edit {
		logger.Error.Println(errors.ErrNotAuthorised)
		c.JSON(http.StatusForbidden, errors.Body{
			Error:  errors.ErrNotAuthorised.Error(),
			Status: errors.StatusNotAuthorised,
		})
		return
	}
	var body editUserBody

	if err := c.ShouldBindJSON(&body); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusBadRequest, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusBadJSON,
		})
		return
	}

	userId := c.Param("userId")
	if match, err := regexp.MatchString("^[0-9]+$", userId); err != nil {
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

	intUserId, err := strconv.ParseInt(userId, 10, 64)
	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	var userExists bool

	if err := db.Db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id=$1)", userId).Scan(&userExists); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	if !userExists {
		logger.Error.Println(errors.ErrUserNotFound)
		c.JSON(http.StatusNotFound, errors.Body{
			Error:  errors.ErrUserNotFound.Error(),
			Status: errors.StatusUserNotFound,
		})
		return
	}

	if body.Password == nil && body.Email == nil && body.Username == nil {
		logger.Error.Println(errors.ErrAllFieldsEmpty)
		c.JSON(http.StatusBadRequest, errors.Body{
			Error:  errors.ErrAllFieldsEmpty.Error(),
			Status: errors.StatusAllFieldsEmpty,
		})
		return
	}
	newUserInfo := events.User{}
	if body.Password != nil {
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

		if _, err = db.Db.Exec("UPDATE users SET password=$1 WHERE id=$2", strHashedPass, userId); err != nil {
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
		if _, err = db.Db.Exec("UPDATE users SET email=$1 WHERE id=$2", *body.Email, userId); err != nil {
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
		if _, err = db.Db.Exec("UPDATE users SET username=$1 WHERE id=$2", *body.Username, userId); err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
		newUserInfo.Name = *body.Username
	}
	if body.Flags != nil {
		if _, err = db.Db.Exec("UPDATE users SET flags=$1 WHERE id=$2", *body.Flags, userId); err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
		*newUserInfo.Flags = *body.Flags
	}
	res := wsclient.DataFrame{
		Op:    wsclient.TYPE_DISPATCH,
		Data:  newUserInfo,
		Event: events.UPDATE_USER_INFO,
	}
	wsclient.Pools.BroadcastClient(intUserId, res)
	c.Status(http.StatusNoContent)
}
