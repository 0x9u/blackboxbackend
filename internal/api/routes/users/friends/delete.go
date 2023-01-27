package friends

import (
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

func Delete(c *gin.Context) {
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
	intUserId, err := strconv.Atoi(userId)
	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	var isBlocked bool
	if err := db.Db.QueryRow(`
		SELECT EXISTS (SELECT 1 FROM blocked WHERE user_id = $1 AND blocked_id = $2)
		 OR 
		EXISTS(SELECT 1 FROM blocked WHERE user_id = $2 AND blocked_id = $1)
		`, user.Id, userId).Scan(&isBlocked); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	if isBlocked {
		logger.Error.Println(errors.ErrFriendBlocked)
		c.JSON(http.StatusBadRequest, errors.Body{
			Error:  errors.ErrFriendBlocked.Error(),
			Status: errors.StatusFriendBlocked,
		})
		return
	}
	var friendExists bool
	if err := db.Db.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM friends WHERE user_id = $1 AND friend_id = $2) OR EXISTS(SELECT 1 FROM friends WHERE user_id = $2 AND friend_id = $1)
		`, user.Id, userId).Scan(&friendExists); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	if !friendExists {
		logger.Error.Println(errors.ErrFriendInvalid)
		c.JSON(http.StatusBadRequest, errors.Body{
			Error:  errors.ErrFriendInvalid.Error(),
			Status: errors.StatusFriendInvalid,
		})
		return
	}
	var isRequested bool
	if err := db.Db.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM friends WHERE user_id = $1 AND friend_id = $2 AND requested = true)
		OR EXISTS(SELECT 1 FROM friends WHERE user_id = $2 AND friend_id = $1 AND requested = true)
		`, user.Id, userId).Scan(&isRequested); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	if _, err := db.Db.Exec(`
		DELETE FROM friends WHERE user_id = $1 AND friend_id = $2
		`, user.Id, userId); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	var typeFriendEvent string
	var typeClientEvent string
	if isRequested {
		typeFriendEvent = events.FRIEND_INCOMING_REQUEST
		typeClientEvent = events.FRIEND_DELETE_REQUEST
	} else {
		typeFriendEvent = events.REMOVE_USER_FRIENDLIST
		typeClientEvent = events.REMOVE_USER_FRIENDLIST
	}
	res := wsclient.DataFrame{
		Op: wsclient.TYPE_DISPATCH,
		Data: events.User{
			UserId: intUserId,
		},
		Event: typeClientEvent,
	}
	resFriend := wsclient.DataFrame{
		Op: wsclient.TYPE_DISPATCH,
		Data: events.User{
			UserId: user.Id,
		},
		Event: typeFriendEvent,
	}
	wsclient.Pools.BroadcastClient(user.Id, res)
	wsclient.Pools.BroadcastClient(intUserId, resFriend)
	c.Status(http.StatusNoContent)
}
