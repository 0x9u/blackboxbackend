package friends

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

	if user.Id == intUserId {
		logger.Error.Println(errors.ErrFriendSelf)
		c.JSON(http.StatusForbidden, errors.Body{
			Error:  errors.ErrFriendSelf.Error(),
			Status: errors.StatusFriendSelf,
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

	var sqlUserImageId sql.NullInt64
	var username string
	if err := db.Db.QueryRow("SELECT files.id, username FROM users LEFT JOIN files ON user_id = users.id WHERE users.id = $1", user.Id).Scan(&sqlUserImageId, &username); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	var userImageId int64
	if sqlUserImageId.Valid {
		userImageId = sqlUserImageId.Int64
	} else {
		userImageId = -1
	}
	var sqlFriendImageId sql.NullInt64
	var friendUsername string
	if err := db.Db.QueryRow("SELECT files.id, username FROM users LEFT JOIN files ON user_id = users.id WHERE users.id = $1", userId).Scan(&sqlFriendImageId, &friendUsername); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	var friendImageId int64
	if sqlFriendImageId.Valid {
		friendImageId = sqlFriendImageId.Int64
	} else {
		friendImageId = -1
	}

	res := wsclient.DataFrame{
		Op: wsclient.TYPE_DISPATCH,
		Data: events.User{
			UserId:  intUserId,
			Name:    friendUsername,
			ImageId: friendImageId,
		},
		Event: events.ADD_FRIEND_REQUEST,
	}
	resFriend := wsclient.DataFrame{
		Op: wsclient.TYPE_DISPATCH,
		Data: events.User{
			UserId:  user.Id,
			Name:    username,
			ImageId: userImageId,
		},
		Event: events.ADD_FRIEND_INCOMING_REQUEST,
	}
	wsclient.Pools.BroadcastClient(user.Id, res)
	wsclient.Pools.BroadcastClient(intUserId, resFriend)

	c.Status(http.StatusCreated)

}

type CreateByNameBody struct {
	Username string `json:"username" binding:"required"`
}

func CreateByName(c *gin.Context) {
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
	var body CreateByNameBody

	if err := c.ShouldBindJSON(&body); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusBadRequest, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusBadRequest,
		})
		return
	}

	var userId int64
	logger.Debug.Println(body.Username)
	if err := db.Db.QueryRow(`
	SELECT id FROM users WHERE username = $1
	`, body.Username).Scan(&userId); err != nil && err != sql.ErrNoRows {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	} else if err == sql.ErrNoRows {
		logger.Error.Println(errors.ErrUserNotFound)
		c.JSON(http.StatusNotFound, errors.Body{
			Error:  errors.ErrUserNotFound.Error(),
			Status: errors.StatusUserNotFound,
		})
		return
	}

	if user.Id == userId {
		logger.Error.Println(errors.ErrFriendSelf)
		c.JSON(http.StatusForbidden, errors.Body{
			Error:  errors.ErrFriendSelf.Error(),
			Status: errors.StatusFriendSelf,
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
	logger.Debug.Println("user id", user.Id, "friend id", userId)
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

	var sqlUserImageId sql.NullInt64
	var username string
	if err := db.Db.QueryRow("SELECT files.id, username FROM users LEFT JOIN files ON user_id = users.id WHERE users.id = $1", user.Id).Scan(&sqlUserImageId, &username); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	var userImageId int64
	if sqlUserImageId.Valid {
		userImageId = sqlUserImageId.Int64
	} else {
		userImageId = -1
	}

	var friendImageId int64
	if err := db.Db.QueryRow("SELECT files FROM files WHERE user_id = $1", userId).Scan(&friendImageId); err != nil && err != sql.ErrNoRows {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	} else if err == sql.ErrNoRows {
		friendImageId = -1
	}

	res := wsclient.DataFrame{
		Op: wsclient.TYPE_DISPATCH,
		Data: events.User{
			UserId:  userId,
			Name:    body.Username,
			ImageId: friendImageId,
		},
		Event: events.ADD_FRIEND_REQUEST,
	}
	resFriend := wsclient.DataFrame{
		Op: wsclient.TYPE_DISPATCH,
		Data: events.User{
			UserId:  user.Id,
			Name:    username,
			ImageId: userImageId,
		},
		Event: events.ADD_FRIEND_INCOMING_REQUEST,
	}
	wsclient.Pools.BroadcastClient(user.Id, res)
	wsclient.Pools.BroadcastClient(userId, resFriend)

	c.Status(http.StatusCreated)
}
