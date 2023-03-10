package msgs

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

func Delete(c *gin.Context) { //deletes message
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

	msgId := c.Param("msgId")
	if match, err := regexp.MatchString("^[0-9]+$", msgId); err != nil {
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
	intMsgId, err := strconv.ParseInt(msgId, 10, 64)
	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	dmId := c.Param("dmId")
	if match, err := regexp.MatchString("^[0-9]+$", dmId); err != nil {
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

	intDmId, err := strconv.ParseInt(dmId, 10, 64)
	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	var msgExists bool
	if err := db.Db.QueryRow("SELECT EXISTS(SELECT 1 FROM directmsgs WHERE id = $1 AND user_id = $2)", msgId, user.Id).Scan(&msgExists); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	if !msgExists {
		logger.Error.Println(errors.ErrNotExists)
		c.JSON(http.StatusNotFound, errors.Body{
			Error:  errors.ErrNotExists.Error(),
			Status: errors.StatusNotExists,
		})
		return
	}

	if _, err = db.Db.Exec("DELETE FROM directmsgs where id = $1", msgId); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	var otherUser int64

	if err = db.Db.QueryRow("SELECT user_id FROM userdirectmsgsguild WHERE dm_id = $1 AND user_id != $2", dmId, user.Id).Scan(&otherUser); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	res := wsclient.DataFrame{
		Op: wsclient.TYPE_DISPATCH,
		Data: events.Msg{
			MsgId: intMsgId,
			DmId:  intDmId,
		},
		Event: events.DELETE_GUILD_MESSAGE,
	}

	wsclient.Pools.BroadcastClient(user.Id, res)
	wsclient.Pools.BroadcastClient(otherUser, res)
	c.Status(http.StatusNoContent)

}
