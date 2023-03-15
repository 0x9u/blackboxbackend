package friends

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

func Get(c *gin.Context) {
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

	rows, err := db.Db.Query(`
		SELECT f.sender_id AS friend_id, u.username AS username  FROM friends f INNER JOIN users u ON f.sender_id = u.id WHERE f.receiver_id = $1 AND f.friended = true
		 UNION
		SELECT f.receiver_id AS friend_id, u.username AS username FROM friends f INNER JOIN users u ON f.receiver_id = u.id WHERE f.sender_id = $1 AND f.friended = true
	`)
	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	defer rows.Close()
	var friends []events.User
	for rows.Next() {
		var friendUser events.User
		if err := rows.Scan(&friendUser.UserId, &friendUser.Name); err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
		friends = append(friends, friendUser)
	}
	c.JSON(http.StatusOK, friends)
}
