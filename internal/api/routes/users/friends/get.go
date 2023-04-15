package friends

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
		SELECT f.sender_id AS friend_id, u.username AS username f.id AS image_id FROM friends f INNER JOIN users u ON f.sender_id = u.id LEFT JOIN files f ON f.user_id = f.sender_id WHERE f.receiver_id = $1 AND f.friended = true
		 UNION
		SELECT f.receiver_id AS friend_id, u.username AS username, f.id AS image_id FROM friends f INNER JOIN users u ON f.receiver_id = u.id LEFT JOIN files f ON f.user_id = f.receiver_id WHERE f.sender_id = $1 AND f.friended = true
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
		var imageId sql.NullInt64
		if err := rows.Scan(&friendUser.UserId, &friendUser.Name, &imageId); err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
		if imageId.Valid {
			friendUser.ImageId = imageId.Int64
		} else {
			friendUser.ImageId = -1
		}
		friends = append(friends, friendUser)
	}
	c.JSON(http.StatusOK, friends)
}
