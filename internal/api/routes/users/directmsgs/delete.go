package directmsgs

import (
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

func Delete(c *gin.Context) {
	user := c.MustGet(middleware.User).(*session.Session)
	if user == nil {
		errors.SendErrorResponse(c, errors.ErrSessionDidntPass, errors.StatusInternalError)
		return
	}

	dmId := c.Param("dmId")
	if match, err := regexp.MatchString("^[0-9]+$", dmId); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	} else if !match {
		errors.SendErrorResponse(c, errors.ErrRouteParamInvalid, errors.StatusRouteParamInvalid)
		return
	}

	intDmId, err := strconv.ParseInt(dmId, 10, 64)
	if err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	var dmExists bool
	if err := db.Db.QueryRow("SELECT EXISTS(SELECT 1 FROM userguilds WHERE user_id = $1 AND guild_id = $2 AND receiver_id IS NOT NULL)", user.Id, intDmId).Scan(&dmExists); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	if !dmExists {
		errors.SendErrorResponse(c, errors.ErrDmNotExist, errors.StatusDmNotExist)
		return
	}

	if _, err := db.Db.Exec("UPDATE userguilds SET left_dm = true WHERE user_id = $1 AND guild_id = $2", user.Id, dmId); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	var userId int64
	var username string
	var sqlImageId sql.NullInt64

	if err := db.Db.QueryRow("SELECT ug.receiver_id, username, f.id FROM users INNER JOIN userguilds ug ON ug.user_id = $1 AND ug.guild_id = $2 LEFT JOIN files f ON f.user_id = ug.receiver_id  WHERE users.id = $1", user.Id, dmId).Scan(&userId, &username, &sqlImageId); err != nil {
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
			DmId: intDmId,
			UserInfo: events.User{
				UserId:  userId,
				Name:    username,
				ImageId: imageId,
			},
			Unread: events.UnreadMsg{},
		},
		Event: events.DM_CREATE,
	}
	wsclient.Pools.BroadcastClient(user.Id, res)
	c.Status(http.StatusNoContent)
}
