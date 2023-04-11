package requests

import (
	"database/sql"
	"net/http"
	"regexp"
	"strconv"

	"github.com/asianchinaboi/backendserver/internal/api/middleware"
	"github.com/asianchinaboi/backendserver/internal/db"
	"github.com/asianchinaboi/backendserver/internal/errors"
	"github.com/asianchinaboi/backendserver/internal/events"
	"github.com/asianchinaboi/backendserver/internal/logger"
	"github.com/asianchinaboi/backendserver/internal/session"
	"github.com/asianchinaboi/backendserver/internal/wsclient"
	"github.com/gin-gonic/gin"
)

func Accept(c *gin.Context) {
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

	intUserId, err := strconv.ParseInt(userId, 10, 64)
	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	var isRequested bool

	if err := db.Db.QueryRow(`
	SELECT EXISTS(SELECT 1 FROM friends WHERE user_id = $1 AND friend_id = $2 AND friended = false)
	 OR 
	EXISTS(SELECT 1 FROM friends WHERE user_id = $2 AND friend_id = $1 AND friended = false)
	`, userId, user.Id).Scan(&isRequested); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	if !isRequested {
		logger.Error.Println(errors.ErrFriendRequestNotFound)
		c.JSON(http.StatusBadRequest, errors.Body{
			Error:  errors.ErrFriendRequestNotFound.Error(),
			Status: errors.StatusFriendRequestNotFound,
		})
		return
	}

	if _, err := db.Db.Exec(`
	UPDATE friends SET friended = true WHERE (user_id = $1 AND friend_id = $2) OR (user_id = $2 AND friend_id = $1)
	`, userId, user.Id); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	var clientUsername string
	var clientImage sql.NullInt64
	if err := db.Db.QueryRow(`
	SELECT username, image_id FROM users WHERE id = $1
	`, user.Id).Scan(&clientUsername, &clientImage); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	var resClientImage int64
	if clientImage.Valid {
		resClientImage = clientImage.Int64
	} else {
		resClientImage = clientImage.Int64
	}

	var friendUsername string
	var friendImage sql.NullInt64
	if err := db.Db.QueryRow(`
	SELECT username, image_id FROM users WHERE id = $1
	`, userId).Scan(&friendUsername, &friendImage); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	var resFriendImage int64
	if friendImage.Valid {
		resFriendImage = friendImage.Int64
	} else {
		resFriendImage = -1
	}

	res := wsclient.DataFrame{
		Op: wsclient.TYPE_DISPATCH,
		Data: events.User{
			UserId: intUserId,
		},
		Event: events.REMOVE_FRIEND_REQUEST,
	}
	resFriend := wsclient.DataFrame{
		Op: wsclient.TYPE_DISPATCH,
		Data: events.User{
			UserId: user.Id,
		},
		Event: events.REMOVE_FRIEND_INCOMING_REQUEST,
	}

	wsclient.Pools.BroadcastClient(user.Id, res)
	wsclient.Pools.BroadcastClient(intUserId, resFriend)

	resAfter := wsclient.DataFrame{
		Op: wsclient.TYPE_DISPATCH,
		Data: events.User{
			UserId:  intUserId,
			Name:    friendUsername,
			ImageId: resFriendImage,
		},
		Event: events.ADD_USER_FRIENDLIST,
	}
	resFriendAfter := wsclient.DataFrame{
		Op: wsclient.TYPE_DISPATCH,
		Data: events.User{
			UserId:  user.Id,
			Name:    clientUsername,
			ImageId: resClientImage,
		},
		Event: events.ADD_USER_FRIENDLIST,
	}

	wsclient.Pools.BroadcastClient(user.Id, resAfter)
	wsclient.Pools.BroadcastClient(intUserId, resFriendAfter)

	c.Status(http.StatusNoContent)

}
