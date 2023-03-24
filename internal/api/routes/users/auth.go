package users

import (
	"database/sql"
	"net/http"

	"github.com/asianchinaboi/backendserver/internal/db"
	"github.com/asianchinaboi/backendserver/internal/errors"
	"github.com/asianchinaboi/backendserver/internal/events"
	"github.com/asianchinaboi/backendserver/internal/logger"
	"github.com/asianchinaboi/backendserver/internal/session"
	"github.com/gin-gonic/gin"
)

func userAuth(c *gin.Context) {
	logger.Debug.Println("Getting user")

	var user events.User

	if err := c.ShouldBindJSON(&user); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusBadRequest, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusBadRequest,
		})
		return
	}
	var userHashedPass string
	if err := db.Db.QueryRow("SELECT password, id FROM users WHERE username = $1", user.Name).Scan(&userHashedPass, &user.UserId); err != nil && err != sql.ErrNoRows {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	} else if err != nil {
		logger.Error.Println(errors.ErrUserNotFound)
		c.JSON(http.StatusNotFound, errors.Body{
			Error:  errors.ErrUserNotFound.Error(),
			Status: errors.StatusUserNotFound,
		})
		return
	}
	if correctPass := comparePasswords(user.Password, userHashedPass); !correctPass {
		logger.Error.Println(errors.ErrInvalidPass)
		c.JSON(http.StatusForbidden, errors.Body{
			Error:  errors.ErrInvalidPass.Error(),
			Status: errors.StatusInvalidPass,
		})
		return
	}

	logger.Debug.Println("Generating token")

	authData, err := session.GenToken(user.UserId)
	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	c.JSON(http.StatusOK, authData)
}
