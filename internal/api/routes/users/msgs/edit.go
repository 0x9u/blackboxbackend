package msgs

import (
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/asianchinaboi/backendserver/internal/api/middleware"
	"github.com/asianchinaboi/backendserver/internal/db"
	"github.com/asianchinaboi/backendserver/internal/errors"
	"github.com/asianchinaboi/backendserver/internal/events"
	"github.com/asianchinaboi/backendserver/internal/logger"
	"github.com/asianchinaboi/backendserver/internal/session"
	"github.com/asianchinaboi/backendserver/internal/wsclient"
	"github.com/gin-gonic/gin"
)

func Edit(c *gin.Context) {
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
	intMsgId, err := strconv.Atoi(msgId)
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

	intDmId, err := strconv.Atoi(dmId)
	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	var msg events.Msg

	if err := c.ShouldBindJSON(&msg); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusBadRequest, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusBadJSON,
		})
		return
	}

	if msg.Content == "" {
		logger.Error.Println(errors.ErrInvalidDetails)
		c.JSON(http.StatusUnprocessableEntity, errors.Body{
			Error:  errors.ErrInvalidDetails.Error(),
			Status: errors.StatusInvalidDetails,
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

	timestamp := time.Now().UnixMilli()

	if _, err = db.Db.Exec("UPDATE directmsgs SET content = $1, modified = $2 WHERE id = $3", msg.Content, timestamp, msgId); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	var otherUser int

	if err := db.Db.QueryRow("SELECT user_id FROM userdirectmsgsguild WHERE dm_id = $1 AND user_id != $2 ", dmId, user.Id).Scan(&otherUser); err != nil {
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
			MsgId:    intMsgId,
			DmId:     intDmId,
			Content:  msg.Content,
			Modified: timestamp,
			Author: events.User{
				UserId: user.Id,
			},
		},
		Event: events.UPDATE_GUILD_MESSAGE,
	}
	wsclient.Pools.BroadcastClient(user.Id, res)
	wsclient.Pools.BroadcastClient(otherUser, res)
	c.Status(http.StatusNoContent)
}
