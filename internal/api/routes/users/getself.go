package users

import (
	"net/http"

	"github.com/asianchinaboi/backendserver/internal/api/middleware"
	"github.com/asianchinaboi/backendserver/internal/db"
	"github.com/asianchinaboi/backendserver/internal/errors"
	"github.com/asianchinaboi/backendserver/internal/events"
	"github.com/asianchinaboi/backendserver/internal/logger"
	"github.com/asianchinaboi/backendserver/internal/session"
	"github.com/gin-gonic/gin"
)

func getSelfInfo(c *gin.Context) {
	user := c.MustGet(middleware.User).(*session.Session)
	var body events.User
	if err := db.Db.QueryRow("SELECT email, username FROM users WHERE id=$1", user.Id).Scan(&body.Email, &body.Name); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	//placeholder for now
	body.Icon = 0
	c.JSON(http.StatusOK, body)
}
