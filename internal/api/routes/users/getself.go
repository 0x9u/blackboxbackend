package users

import (
	"database/sql"
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
	var imageId sql.NullInt64
	if err := db.Db.QueryRow("SELECT users.id email, username, files.id, options FROM users LEFT JOIN files ON files.user_id = users.id WHERE users.id=$1", user.Id).Scan(&body.UserId, &body.Email, &body.Name, &imageId, &body.Options); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	if imageId.Valid {
		body.ImageId = imageId.Int64
	} else {
		body.ImageId = -1
	}
	//placeholder for now
	c.JSON(http.StatusOK, body)
}
