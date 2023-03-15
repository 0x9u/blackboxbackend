package directmsgs

import (
	"net/http"

	"github.com/asianchinaboi/backendserver/internal/api/middleware"
	"github.com/asianchinaboi/backendserver/internal/db"
	"github.com/asianchinaboi/backendserver/internal/errors"
	"github.com/asianchinaboi/backendserver/internal/events"
	"github.com/asianchinaboi/backendserver/internal/logger"
	"github.com/asianchinaboi/backendserver/internal/session"
	"github.com/asianchinaboi/backendserver/internal/uid"
	"github.com/asianchinaboi/backendserver/internal/wsclient"
	"github.com/gin-gonic/gin"
)

type CreateDmBody struct {
	ReceiverId int64 `json:"receiverId"`
}

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

	var body CreateDmBody

	if err := c.ShouldBindJSON(&body); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusBadRequest, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusBadJSON,
		})
		return
	}

	var userExists bool

	if err := db.Db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", body.ReceiverId).Scan(&userExists); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	if !userExists {
		logger.Error.Println(errors.ErrUserNotFound)
		c.JSON(http.StatusNotFound, errors.Body{
			Error:  errors.ErrUserNotFound.Error(),
			Status: errors.StatusUserNotFound,
		})
	}

	var dmExists bool

	if err := db.Db.QueryRow("SELECT EXISTS(SELECT 1 FROM usersdirectmsgsguild WHERE user_id = $1 AND dm_id = (SELECT dm_id FROM usersdirectmsgsguild WHERE user_id = $2))", user.Id, body.ReceiverId).Scan(&dmExists); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	if dmExists {
		logger.Error.Println(errors.ErrDmAlreadyExists)
		c.JSON(http.StatusConflict, errors.Body{
			Error:  errors.ErrDmAlreadyExists.Error(),
			Status: errors.StatusDmAlreadyExists,
		})
		return
	}

	dmId := uid.Snowflake.Generate().Int64()

	if _, err := db.Db.Exec("INSERT INTO directmsgsguild VALUES ($1)", dmId); err != nil { //make new dm identity
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	if _, err := db.Db.Exec("INSERT INTO usersdirectmsgsguild VALUES ($1, $2, false), ($1, $3, true)", dmId, user.Id, body.ReceiverId); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	var username string

	if err := db.Db.QueryRow("SELECT username FROM users WHERE id = $1", user.Id).Scan(&username); err != nil {
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
			UserId: body.ReceiverId,
			Name:   username,
			Icon:   0,
		},
		Event: events.CREATE_DM,
	}
	wsclient.Pools.BroadcastClient(user.Id, res)

	c.Status(http.StatusCreated)
}
