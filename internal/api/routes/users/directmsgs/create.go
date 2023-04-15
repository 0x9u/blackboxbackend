package directmsgs

import (
	"context"
	"database/sql"
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
			Status: errors.StatusBadRequest,
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

	if err := db.Db.QueryRow("SELECT EXISTS(SELECT 1 FROM userguilds WHERE user_id = $1 AND receiver_id = $2)", user.Id, body.ReceiverId).Scan(&dmExists); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	if dmExists {
		if _, err := db.Db.Exec("UPDATE userguilds SET left_dm = false WHERE user_id = $1 AND receiver_id = $2", user.Id, body.ReceiverId); err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
		var username string
		var sqlImageId sql.NullInt64

		if err := db.Db.QueryRow("SELECT username, f.id FROM users LEFT JOIN files f ON f.user_id = users.id WHERE users.id = $1", user.Id).Scan(&username, &sqlImageId); err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}

		var imageId int64
		if sqlImageId.Valid {
			imageId = sqlImageId.Int64
		} else {
			imageId = -1
		}

		res := wsclient.DataFrame{
			Op: wsclient.TYPE_DISPATCH,
			Data: events.User{
				UserId:  body.ReceiverId,
				Name:    username,
				ImageId: imageId,
			},
			Event: events.CREATE_DM,
		}
		wsclient.Pools.BroadcastClient(user.Id, res)

		c.Status(http.StatusCreated)
		return
	}
	//BEGIN TRANSACTION
	ctx := context.Background()
	tx, err := db.Db.BeginTx(ctx, nil)
	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	defer tx.Rollback() //rollback changes if failed

	dmId := uid.Snowflake.Generate().Int64()

	if _, err := tx.ExecContext(ctx, "INSERT INTO guilds (id, dm) VALUES ($1, true)", dmId); err != nil { //make new dm identity
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	if _, err := tx.ExecContext(ctx, "INSERT INTO userguilds(guild_id, user_id, receiver_id, left_dm, owner) VALUES ($1, $2, $3, false, true), ($1, $3, $2, true, true)", dmId, user.Id, body.ReceiverId); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	if _, err := tx.ExecContext(ctx, "INSERT INTO unreadmsgs(guild_id, user_id) VALUES ($1, $2), ($1, $3)", dmId, user.Id, body.ReceiverId); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	var username string
	var sqlImageId sql.NullInt64

	if err := db.Db.QueryRow("SELECT username, f.id FROM users LEFT JOIN files f ON f.user_id = users.id WHERE users.id = $1", user.Id).Scan(&username, &sqlImageId); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	var imageId int64
	if sqlImageId.Valid {
		imageId = sqlImageId.Int64
	} else {
		imageId = -1
	}

	if err := tx.Commit(); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
	}

	res := wsclient.DataFrame{
		Op: wsclient.TYPE_DISPATCH,
		Data: events.User{
			UserId:  body.ReceiverId,
			Name:    username,
			ImageId: imageId,
		},
		Event: events.CREATE_DM,
	}
	wsclient.Pools.BroadcastClient(user.Id, res)

	c.Status(http.StatusCreated)
}
