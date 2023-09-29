package directmsgs

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/asianchinaboi/backendserver/internal/api/middleware"
	"github.com/asianchinaboi/backendserver/internal/db"
	"github.com/asianchinaboi/backendserver/internal/errors"
	"github.com/asianchinaboi/backendserver/internal/events"
	"github.com/asianchinaboi/backendserver/internal/session"
	"github.com/asianchinaboi/backendserver/internal/uid"
	"github.com/asianchinaboi/backendserver/internal/wsclient"
	"github.com/gin-gonic/gin"
)

type CreateDmBody struct {
	ReceiverId int64 `json:"receiverId,string" binding:"required"`
}

func Create(c *gin.Context) {
	user := c.MustGet(middleware.User).(*session.Session)
	if user == nil {
		errors.SendErrorResponse(c, errors.ErrSessionDidntPass, errors.StatusInternalError)
		return
	}

	var body CreateDmBody

	if err := c.ShouldBindJSON(&body); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusBadRequest)
		return
	}

	if body.ReceiverId == user.Id {
		errors.SendErrorResponse(c, errors.ErrDmCannotDmSelf, errors.StatusDmCannotDmSelf)
		return
	}

	var userExists bool

	if err := db.Db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", body.ReceiverId).Scan(&userExists); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	if !userExists {
		errors.SendErrorResponse(c, errors.ErrUserNotFound, errors.StatusUserNotFound)
		return
	}

	var dmExists bool

	if err := db.Db.QueryRow("SELECT EXISTS(SELECT 1 FROM userguilds WHERE user_id = $1 AND receiver_id = $2)", user.Id, body.ReceiverId).Scan(&dmExists); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	if dmExists {
		var dmId int64
		//will return two rows but query row should be fine
		if err := db.Db.QueryRow("UPDATE userguilds SET left_dm = false WHERE user_id = $1 AND receiver_id = $2 RETURNING guild_id", user.Id, body.ReceiverId).Scan(&dmId); err != nil {
			errors.SendErrorResponse(c, err, errors.StatusInternalError)
			return
		}
		var username string
		var sqlImageId sql.NullInt64

		if err := db.Db.QueryRow("SELECT username, f.id FROM users LEFT JOIN files f ON f.user_id = users.id WHERE users.id = $1", user.Id).Scan(&username, &sqlImageId); err != nil {
			errors.SendErrorResponse(c, err, errors.StatusInternalError)
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
			Data: events.Dm{
				DmId: dmId,
				UserInfo: events.User{
					UserId:  body.ReceiverId,
					Name:    username,
					ImageId: imageId,
				},
				Unread: events.UnreadMsg{},
			},
			Event: events.DM_CREATE,
		}
		wsclient.Pools.BroadcastClient(user.Id, res)

		c.Status(http.StatusCreated)
		return
	}
	//BEGIN TRANSACTION
	ctx := context.Background()
	tx, err := db.Db.BeginTx(ctx, nil)
	if err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}
	defer tx.Rollback() //rollback changes if failed

	dmId := uid.Snowflake.Generate().Int64()

	if _, err := tx.ExecContext(ctx, "INSERT INTO guilds (id, dm) VALUES ($1, true)", dmId); err != nil { //make new dm identity
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	if _, err := tx.ExecContext(ctx, "INSERT INTO userguilds(guild_id, user_id, receiver_id, left_dm, owner) VALUES ($1, $2, $3, false, true), ($1, $3, $2, true, true)", dmId, user.Id, body.ReceiverId); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	if _, err := tx.ExecContext(ctx, "INSERT INTO unreadmsgs(guild_id, user_id) VALUES ($1, $2), ($1, $3)", dmId, user.Id, body.ReceiverId); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	var username string
	var sqlImageId sql.NullInt64

	if err := db.Db.QueryRow("SELECT username, f.id FROM users LEFT JOIN files f ON f.user_id = users.id WHERE users.id = $1", user.Id).Scan(&username, &sqlImageId); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	var imageId int64
	if sqlImageId.Valid {
		imageId = sqlImageId.Int64
	} else {
		imageId = -1
	}

	if err := tx.Commit(); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	res := wsclient.DataFrame{
		Op: wsclient.TYPE_DISPATCH,
		Data: events.Dm{
			DmId: dmId,
			UserInfo: events.User{
				UserId:  body.ReceiverId,
				Name:    username,
				ImageId: imageId,
			},
			Unread: events.UnreadMsg{},
		},
		Event: events.DM_CREATE,
	}
	wsclient.Pools.BroadcastClient(user.Id, res)

	c.Status(http.StatusCreated)
}
