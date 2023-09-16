package requests

import (
	"context"
	"database/sql"
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

func Accept(c *gin.Context) {
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

	var isRequested bool

	if err := db.Db.QueryRow(`
	SELECT EXISTS(SELECT 1 FROM friends WHERE user_id = $1 AND friend_id = $2 AND friended = false)
	`, userId, user.Id).Scan(&isRequested); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	if !isRequested {
		errors.SendErrorResponse(c, errors.ErrFriendRequestNotFound, errors.StatusFriendRequestNotFound)
		return
	}

	ctx := context.Background()
	tx, err := db.Db.BeginTx(ctx, nil)
	if err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
	UPDATE friends SET friended = true WHERE (user_id = $1 AND friend_id = $2)
	`, userId, user.Id); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	if _, err := tx.ExecContext(ctx, `
	INSERT INTO friends(user_id, friend_id, friended) VALUES ($1, $2, true)
`, user.Id, userId); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	if err := tx.Commit(); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	var clientUsername string
	var clientImage sql.NullInt64
	if err := db.Db.QueryRow(`
	SELECT username, f.id FROM users LEFT JOIN files f ON f.user_id = users.id WHERE users.id = $1
	`, user.Id).Scan(&clientUsername, &clientImage); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	var resClientImage int64
	if clientImage.Valid {
		resClientImage = clientImage.Int64
	} else {
		resClientImage = -1
	}

	var friendUsername string
	var friendImage sql.NullInt64
	if err := db.Db.QueryRow(`
	SELECT username, f.id FROM users LEFT JOIN files f ON f.user_id = users.id WHERE users.id = $1
	`, userId).Scan(&friendUsername, &friendImage); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
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
