package blocked

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

func Create(c *gin.Context) {
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

	var isBlocked bool
	if err := db.Db.QueryRow(`
		SELECT EXISTS (SELECT 1 FROM blocked WHERE user_id = $1 AND blocked_id = $2)
		`, user.Id, userId).Scan(&isBlocked); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}
	if isBlocked {
		errors.SendErrorResponse(c, errors.ErrFriendBlocked, errors.StatusFriendBlocked)
		return
	}

	var isFriends bool
	if err := db.Db.QueryRow(`
		SELECT EXISTS (SELECT 1 FROM friends WHERE user_id = $1 AND friend_id = $2)
		`, user.Id, userId).Scan(&isFriends); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
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
		INSERT INTO blocked (user_id, blocked_id) VALUES ($1, $2)
		`, user.Id, userId); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}
	if isFriends {
		if _, err := tx.ExecContext(ctx, `
		DELETE FROM friends WHERE (user_id = $1 AND friend_id = $2) OR (user_id = $2 AND friend_id = $1)
		`, user.Id, userId); err != nil {
			errors.SendErrorResponse(c, err, errors.StatusInternalError)
			return
		}
	}

	var hasBeenRequested bool
	var isTheRequestor bool

	if err := db.Db.QueryRow(`
	SELECT EXISTS(SELECT 1 FROM friends WHERE (user_id = $1 AND friend_id = $2) AND friended = false), EXISTS(SELECT 1 FROM friends WHERE (user_id = $2 AND friend_id = $1) AND friended = false)
	`, userId, user.Id).Scan(&hasBeenRequested, &isTheRequestor); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	if isTheRequestor || hasBeenRequested {
		if _, err := db.Db.Exec(`
		DELETE FROM friends WHERE (user_id = $1 AND friend_id = $2) OR (user_id = $2 AND friend_id = $1)
		`, userId, user.Id); err != nil {
			errors.SendErrorResponse(c, err, errors.StatusInternalError)
			return
		}
	}

	var sqlBlockedImageId sql.NullInt64
	var blockedUsername string
	if err := db.Db.QueryRow("SELECT username, f.id FROM users LEFT JOIN files f ON f.user_id = users.id WHERE users.id = $1", userId).Scan(&blockedUsername, &sqlBlockedImageId); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	var blockedImageId int64
	if sqlBlockedImageId.Valid {
		blockedImageId = sqlBlockedImageId.Int64
	} else {
		blockedImageId = -1
	}

	if err := tx.Commit(); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}
	res := wsclient.DataFrame{
		Op: wsclient.TYPE_DISPATCH,
		Data: events.User{
			UserId:  intUserId,
			Name:    blockedUsername,
			ImageId: blockedImageId,
		},
		Event: events.USER_BLOCKED_ADD,
	}
	wsclient.Pools.BroadcastClient(user.Id, res)
	if isFriends {
		resAfter := wsclient.DataFrame{
			Op: wsclient.TYPE_DISPATCH,
			Data: events.User{
				UserId: intUserId,
			},
			Event: events.USER_FRIEND_REMOVE,
		}
		resFriendAfter := wsclient.DataFrame{
			Op: wsclient.TYPE_DISPATCH,
			Data: events.User{
				UserId: user.Id,
			},
			Event: events.USER_FRIEND_REMOVE,
		}
		wsclient.Pools.BroadcastClient(user.Id, resFriendAfter)
		wsclient.Pools.BroadcastClient(intUserId, resAfter)
	}

	if hasBeenRequested || isTheRequestor {
		var eventTypeRes string
		var eventTypeResFriend string
		if isTheRequestor {
			eventTypeRes = events.USER_FRIEND_REQUEST_REMOVE
			eventTypeResFriend = events.USER_FRIEND_INCOMING_REQUEST_REMOVE
		} else {
			eventTypeRes = events.USER_FRIEND_INCOMING_REQUEST_REMOVE
			eventTypeResFriend = events.USER_FRIEND_REQUEST_REMOVE
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

	}
	c.Status(http.StatusNoContent)
}
