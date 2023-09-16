package friends

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

func Delete(c *gin.Context) {
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

	var friendExists bool
	if err := db.Db.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM friends WHERE user_id = $1 AND friend_id = $2 AND friended = true)
		`, user.Id, userId).Scan(&friendExists); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}
	if !friendExists {
		errors.SendErrorResponse(c, errors.ErrFriendInvalid, errors.StatusFriendInvalid)
		return
	}

	if _, err := db.Db.Exec(`
		DELETE FROM friends WHERE (user_id = $1 AND friend_id = $2) OR (user_id = $2 AND friend_id = $1)
		`, user.Id, userId); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	res := wsclient.DataFrame{
		Op: wsclient.TYPE_DISPATCH,
		Data: events.User{
			UserId: intUserId,
		},
		Event: events.REMOVE_USER_FRIENDLIST,
	}
	resFriend := wsclient.DataFrame{
		Op: wsclient.TYPE_DISPATCH,
		Data: events.User{
			UserId: user.Id,
		},
		Event: events.REMOVE_USER_FRIENDLIST,
	}
	wsclient.Pools.BroadcastClient(user.Id, res)
	wsclient.Pools.BroadcastClient(intUserId, resFriend)
	c.Status(http.StatusNoContent)
}
