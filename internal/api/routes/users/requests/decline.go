package requests

import (
	"net/http"
	"regexp"
	"strconv"

	"github.com/asianchinaboi/backendserver/internal/api/middleware"
	"github.com/asianchinaboi/backendserver/internal/db"
	"github.com/asianchinaboi/backendserver/internal/errors"
	"github.com/asianchinaboi/backendserver/internal/events"
	"github.com/asianchinaboi/backendserver/internal/session"
	"github.com/asianchinaboi/backendserver/internal/wsclient"
	"github.com/gin-gonic/gin"
)

func Decline(c *gin.Context) { //original sender of request can also decline
	user := c.MustGet(middleware.User).(*session.Session)
	if user == nil {
		errors.SendErrorResponse(c, errors.ErrSessionDidntPass, errors.StatusInternalError)
		return
	}

	userId := c.Param("userId")
	if match, err := regexp.MatchString("^[0-9]+$", userId); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	} else if !match {
		errors.SendErrorResponse(c, errors.ErrRouteParamInvalid, errors.StatusRouteParamInvalid)
		return
	}

	intUserId, err := strconv.ParseInt(userId, 10, 64)
	if err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	var hasBeenRequested bool
	var isTheRequestor bool

	if err := db.Db.QueryRow(`
	SELECT EXISTS(SELECT 1 FROM friends WHERE (user_id = $1 AND friend_id = $2) AND friended = false), EXISTS(SELECT 1 FROM friends WHERE (user_id = $2 AND friend_id = $1) AND friended = false)
	`, userId, user.Id).Scan(&hasBeenRequested, &isTheRequestor); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	if !isTheRequestor && !hasBeenRequested {
		errors.SendErrorResponse(c, errors.ErrFriendRequestNotFound, errors.StatusFriendRequestNotFound)
		return
	}

	if _, err := db.Db.Exec(`
	DELETE FROM friends WHERE (user_id = $1 AND friend_id = $2) OR (user_id = $2 AND friend_id = $1)
	`, userId, user.Id); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	var eventTypeRes string
	var eventTypeResFriend string
	if isTheRequestor {
		eventTypeRes = events.REMOVE_FRIEND_REQUEST
		eventTypeResFriend = events.REMOVE_FRIEND_INCOMING_REQUEST
	} else {
		eventTypeRes = events.REMOVE_FRIEND_INCOMING_REQUEST
		eventTypeResFriend = events.REMOVE_FRIEND_REQUEST
	}

	res := wsclient.DataFrame{
		Op: wsclient.TYPE_DISPATCH,
		Data: events.User{
			UserId: intUserId,
		},
		Event: eventTypeRes,
	}
	resFriend := wsclient.DataFrame{
		Op: wsclient.TYPE_DISPATCH,
		Data: events.User{
			UserId: user.Id,
		},
		Event: eventTypeResFriend,
	}

	wsclient.Pools.BroadcastClient(user.Id, res)
	wsclient.Pools.BroadcastClient(intUserId, resFriend)

	c.Status(http.StatusNoContent)

}
