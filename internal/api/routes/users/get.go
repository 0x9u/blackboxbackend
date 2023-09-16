package users

import (
	"database/sql"
	"net/http"
	"regexp"

	"github.com/asianchinaboi/backendserver/internal/api/middleware"
	"github.com/asianchinaboi/backendserver/internal/db"
	"github.com/asianchinaboi/backendserver/internal/errors"
	"github.com/asianchinaboi/backendserver/internal/events"
	"github.com/asianchinaboi/backendserver/internal/session"
	"github.com/gin-gonic/gin"
)

func getUserInfo(c *gin.Context) {
	user := c.MustGet(middleware.User).(*session.Session)
	if user == nil {
		errors.SendErrorResponse(c, errors.ErrSessionDidntPass, errors.StatusInternalError)
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

	var userBody events.User
	var imageId sql.NullInt64

	if err := db.Db.QueryRow("SELECT users.id, username, flags, files.id, options FROM users LEFT JOIN files ON files.user_id = users.id WHERE users.id = $1", userId).Scan(&userBody.UserId, &userBody.Name, &userBody.Flags, &imageId, &userBody.Options); err != nil && err == sql.ErrNoRows {
		errors.SendErrorResponse(c, errors.ErrUserNotFound, errors.StatusUserNotFound)
		return
	} else if err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	if imageId.Valid {
		userBody.ImageId = imageId.Int64
	} else {
		userBody.ImageId = -1
	}

	c.JSON(http.StatusOK, userBody)
}

func getUserByUsername(c *gin.Context) {
	user := c.MustGet(middleware.User).(*session.Session)
	if user == nil {
		errors.SendErrorResponse(c, errors.ErrSessionDidntPass, errors.StatusInternalError)
		return
	}

	username := c.Param("username")
	if match, err := regexp.MatchString("^[A-Za-z0-9_]+$", username); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	} else if !match {
		errors.SendErrorResponse(c, errors.ErrRouteParamInvalid, errors.StatusRouteParamInvalid)
		return
	}

	var userBody events.User
	var imageId sql.NullInt64

	if err := db.Db.QueryRow("SELECT users.id, username, flags, files.id, options FROM users LEFT JOIN files ON files.user_id = users.id WHERE username = $1", username).Scan(&userBody.UserId, &userBody.Name, &userBody.Flags, &imageId, &userBody.Options); err != nil && err == sql.ErrNoRows {
		errors.SendErrorResponse(c, errors.ErrUserNotFound, errors.StatusUserNotFound)
		return
	} else if err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	if imageId.Valid {
		userBody.ImageId = imageId.Int64
	} else {
		userBody.ImageId = -1
	}

	c.JSON(http.StatusOK, userBody)
}
