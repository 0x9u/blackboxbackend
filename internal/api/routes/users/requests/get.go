package requests

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

type FriendRequest struct {
	Requested []events.User `json:"requested"` //people you have requested to be friends with
	Pending   []events.User `json:"pending"`   //people who have requested to be friends with you
}

func Get(c *gin.Context) {
	user := c.MustGet(middleware.User).(*session.Session)
	if user == nil {
		errors.SendErrorResponse(c, errors.ErrSessionDidntPass, errors.StatusInternalError)
		return
	}

	pendingReqs, err := db.Db.Query(`
	SELECT f.user_id AS friend_id, u.username AS username, files.id AS image_id FROM friends f INNER JOIN users u ON f.user_id = u.id LEFT JOIN files ON files.user_id = f.user_id WHERE f.friend_id = $1 AND f.friended = false
	`, user.Id)
	if err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}
	defer pendingReqs.Close()
	var requests FriendRequest

	requests.Pending = []events.User{}
	requests.Requested = []events.User{}

	for pendingReqs.Next() {
		var friendUser events.User
		var imageId sql.NullInt64

		if err := pendingReqs.Scan(&friendUser.UserId, &friendUser.Name, &imageId); err != nil {
			errors.SendErrorResponse(c, err, errors.StatusInternalError)
			return
		}
		if imageId.Valid {
			friendUser.ImageId = imageId.Int64
		} else {
			friendUser.ImageId = -1
		}
		requests.Pending = append(requests.Pending, friendUser)
	}

	requestedReqs, err := db.Db.Query(`
	SELECT f.friend_id AS friend_id, u.username AS username, files.id AS image_id FROM friends f INNER JOIN users u ON f.friend_id = u.id LEFT JOIN files ON files.user_id = f.friend_id WHERE f.user_id = $1 AND f.friended = false
	`, user.Id)
	if err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}
	defer requestedReqs.Close()
	for requestedReqs.Next() {
		var friendUser events.User
		var imageId sql.NullInt64
		if err := requestedReqs.Scan(&friendUser.UserId, &friendUser.Name, &imageId); err != nil {
			errors.SendErrorResponse(c, err, errors.StatusInternalError)
			return
		}
		if imageId.Valid {
			friendUser.ImageId = imageId.Int64
		} else {
			friendUser.ImageId = -1
		}
		requests.Requested = append(requests.Requested, friendUser)
	}

	c.JSON(http.StatusOK, requests)
}
