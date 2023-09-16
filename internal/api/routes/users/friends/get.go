package friends

import (
	"database/sql"
	"net/http"

	"github.com/asianchinaboi/backendserver/internal/api/middleware"
	"github.com/asianchinaboi/backendserver/internal/db"
	"github.com/asianchinaboi/backendserver/internal/errors"
	"github.com/asianchinaboi/backendserver/internal/events"
	"github.com/asianchinaboi/backendserver/internal/session"
	"github.com/gin-gonic/gin"
)

func Get(c *gin.Context) {
	user := c.MustGet(middleware.User).(*session.Session)
	if user == nil {
		errors.SendErrorResponse(c, errors.ErrSessionDidntPass, errors.StatusInternalError)
		return
	}

	rows, err := db.Db.Query(`
		SELECT f.friend_id AS friend_id, u.username AS username, files.id AS image_id FROM friends f INNER JOIN users u ON f.friend_id = u.id LEFT JOIN files ON files.user_id = f.friend_id WHERE f.user_id = $1 AND f.friended = true
	`, user.Id)
	if err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}
	defer rows.Close()
	friends := []events.User{}
	for rows.Next() {
		var friendUser events.User
		var imageId sql.NullInt64
		if err := rows.Scan(&friendUser.UserId, &friendUser.Name, &imageId); err != nil {
			errors.SendErrorResponse(c, err, errors.StatusInternalError)
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
