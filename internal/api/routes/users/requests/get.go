package requests

import (
	"database/sql"
	"net/http"
	"regexp"

	"github.com/asianchinaboi/backendserver/internal/api/middleware"
	"github.com/asianchinaboi/backendserver/internal/db"
	"github.com/asianchinaboi/backendserver/internal/errors"
	"github.com/asianchinaboi/backendserver/internal/events"
	"github.com/asianchinaboi/backendserver/internal/logger"
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
		logger.Error.Println("user token not sent in data")
		c.JSON(http.StatusInternalServerError,
			errors.Body{
				Error:  errors.ErrSessionDidntPass.Error(),
				Status: errors.StatusInternalError,
			})
		return
	}

	userId := c.Param("userId")
	if match, err := regexp.MatchString("^[0-9]+$", userId); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	} else if !match {
		logger.Error.Println(errors.ErrRouteParamInvalid)
		c.JSON(http.StatusBadRequest, errors.Body{
			Error:  errors.ErrRouteParamInvalid.Error(),
			Status: errors.StatusRouteParamInvalid,
		})
		return
	}

	pendingReqs, err := db.Db.Query(`
	SELECT f.receiver_id AS friend_id, u.username AS username, files.id AS image_id FROM friends f INNER JOIN users u ON f.receiver_id = u.id LEFT JOIN files ON files.user_id = f.receiver_id WHERE f.sender_id = $1 AND f.friended = false
	`, user.Id)
	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	defer pendingReqs.Close()
	var requests FriendRequest
	for pendingReqs.Next() {
		var friendUser events.User
		var imageId sql.NullInt64

		if err := pendingReqs.Scan(&friendUser.UserId, &friendUser.Name, &imageId); err != nil {
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
		requests.Pending = append(requests.Pending, friendUser)
	}
	requestedReqs, err := db.Db.Query(`
	SELECT f.sender_id AS friend_id, u.username AS username, files.id AS image_id FROM friends f INNER JOIN users u ON f.sender_id = u.id LEFT JOIN files ON files.user_id = f.sender_id WHERE f.receiver_id = $1 AND f.friended = false
	`, user.Id)
	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	defer requestedReqs.Close()
	for requestedReqs.Next() {
		var friendUser events.User
		var imageId sql.NullInt64
		if err := requestedReqs.Scan(&friendUser.UserId, &friendUser.Name, &imageId); err != nil {
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
		requests.Requested = append(requests.Requested, friendUser)
	}

	c.JSON(http.StatusOK, requests)
}
