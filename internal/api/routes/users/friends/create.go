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

func Create(c *gin.Context) {
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

	var isBlocked bool
	var isAlreadyFriends bool
	var isRequested bool

	if err := db.Db.QueryRow(`
		SELECT EXISTS (SELECT 1 FROM blocked WHERE user_id = $1 AND blocked_id = $2)
		 OR 
		EXISTS (SELECT 1 FROM blocked WHERE user_id = $2 AND blocked_id = $1)
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

	if err := db.Db.QueryRow(`
	SELECT EXISTS(SELECT 1 FROM friends WHERE user_id = $1 AND friend_id = $2 AND friended = false) OR EXISTS(SELECT 1 FROM friends WHERE user_id = $2 AND friend_id = $1 AND friended = false)
	`, userId, user.Id).Scan(&isRequested); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	if isRequested {
		logger.Error.Println(errors.ErrFriendAlreadyRequested)
		c.JSON(http.StatusBadRequest, errors.Body{
			Error:  errors.ErrFriendAlreadyRequested.Error(),
			Status: errors.StatusFriendAlreadyRequested,
		})
		return
	}

	if err := db.Db.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM friends WHERE user_id = $1 AND friend_id = $2) OR EXISTS(SELECT 1 FROM friends WHERE user_id = $2 AND friend_id = $1)
		`, user.Id, userId).Scan(&isAlreadyFriends); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	if isAlreadyFriends {
		logger.Error.Println(errors.ErrFriendAlreadyFriends)
		c.JSON(http.StatusBadRequest, errors.Body{
			Error:  errors.ErrFriendAlreadyFriends.Error(),
			Status: errors.StatusFriendAlreadyFriends,
		})
		return
	}

	var sameChat bool
	var friendsOfFriends bool
	var options int

	if err := db.Db.QueryRow(`
	SELECT EXISTS(SELECT 1 FROM userguilds uga WHERE uga.guild_id IN (SELECT guild_id FROM userguilds ugb WHERE ugb.user_id = $2) AND uga.user_id = $1) as samechat,
	 EXISTS(SELECT 1 FROM friends WHERE (user_id = $1 OR friend_id = $1) AND 
	 ( user_id IN (SELECT user_id FROM friends WHERE friend_id = $2) OR friend_id IN (SELECT friend_id FROM friends WHERE user_id = $2))) as friendsoffriends
	`, user.Id, userId).Scan(&sameChat, &friendsOfFriends); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	if err := db.Db.QueryRow(`
		SELECT options FROM users WHERE id = $1
	`, userId).Scan(&options); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	if !((options&events.OAllowFriendsOfFriends) != 0 && friendsOfFriends) &&
		!((options&events.OAllowSameChat) != 0 && sameChat) &&
		!((options & events.OAllowRequestEveryone) != 0) {
		logger.Error.Println(errors.ErrFriendCannotRequest)
		c.JSON(http.StatusForbidden, errors.Body{
			Error:  errors.ErrFriendCannotRequest.Error(),
			Status: errors.StatusFriendCannotRequest,
		})
		return
	}

	if _, err := db.Db.Exec(`
		INSERT INTO friends(user_id, friend_id, friended) VALUES($1, $2, false)
		`, user.Id, userId); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	res := wsclient.DataFrame{
		Op: wsclient.TYPE_DISPATCH,
		Data: events.User{
			UserId: intUserId,
		},
		Event: events.ADD_FRIEND_REQUEST,
	}
	resFriend := wsclient.DataFrame{
		Op: wsclient.TYPE_DISPATCH,
		Data: events.User{
			UserId: user.Id,
		},
		Event: events.ADD_FRIEND_INCOMING_REQUEST,
	}
	wsclient.Pools.BroadcastClient(user.Id, res)
	wsclient.Pools.BroadcastClient(intUserId, resFriend)

	c.Status(http.StatusCreated)

}
