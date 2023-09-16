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
		errors.SendErrorResponse(c, err, errors.StatusBadRequest)
		return
	}
	var userHashedPass string
	if err := db.Db.QueryRow("SELECT password, id FROM users WHERE username = $1", user.Name).Scan(&userHashedPass, &user.UserId); err != nil && err != sql.ErrNoRows {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	} else if err != nil {
		errors.SendErrorResponse(c, errors.ErrUserNotFound, errors.StatusUserNotFound)
		return
	}
	if correctPass := comparePasswords(user.Password, userHashedPass); !correctPass {
		errors.SendErrorResponse(c, errors.ErrInvalidPass, errors.StatusInvalidPass)
		return
	}

	authData, err := session.GenToken(user.UserId)
	if err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}
	c.JSON(http.StatusOK, authData)
}
