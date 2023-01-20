package users

import (
	"net/http"

	"github.com/asianchinaboi/backendserver/internal/db"
	"github.com/asianchinaboi/backendserver/internal/errors"
	"github.com/asianchinaboi/backendserver/internal/events"
	"github.com/asianchinaboi/backendserver/internal/logger"
	"github.com/asianchinaboi/backendserver/internal/session"
	"github.com/gin-gonic/gin"
)

func userCreate(c *gin.Context) {
	logger.Debug.Println("Creating User")
	var user events.User
	if err := c.ShouldBindJSON(&user); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusBadRequest, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusBadJSON,
		})
		return
	}

	logger.Debug.Println("password:", user.Password)

	//validation

	if statusCode, err := events.ValidateUserInput(user); err != nil && statusCode != errors.StatusInternalError {
		logger.Error.Println(err)
		c.JSON(http.StatusUnprocessableEntity, errors.Body{
			Error:  err.Error(),
			Status: statusCode,
		})
		return
	} else if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	var isUsernameTaken bool

	if err := db.Db.QueryRow("SELECT EXISTS (SELECT 1 FROM users WHERE username=$1)", user.Name).Scan(&isUsernameTaken); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	if isUsernameTaken {
		logger.Error.Println(errors.ErrUsernameExists)
		c.JSON(http.StatusConflict, errors.Body{
			Error:  errors.ErrUsernameExists.Error(),
			Status: errors.StatusUsernameExists,
		})
		return
	}
	hashedPass := hashPass(user.Password)

	if err := db.Db.QueryRow("INSERT INTO users (email, password, username) VALUES ($1, $2, $3) RETURNING id", user.Email, hashedPass, user.Name).Scan(&user.UserId); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	//add user role

	if _, err := db.Db.Exec("INSERT INTO userroles (user_id, role_id) VALUES ($1, 1)", user.UserId); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	//create session for new user
	authData, err := session.GenToken(user.UserId)
	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	logger.Debug.Printf("info about new user %v\n", authData)
	c.JSON(http.StatusOK, authData)
}
