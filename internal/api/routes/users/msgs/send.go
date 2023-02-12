package msgs

import (
	"html"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/asianchinaboi/backendserver/internal/api/middleware"
	"github.com/asianchinaboi/backendserver/internal/config"
	"github.com/asianchinaboi/backendserver/internal/db"
	"github.com/asianchinaboi/backendserver/internal/errors"
	"github.com/asianchinaboi/backendserver/internal/events"
	"github.com/asianchinaboi/backendserver/internal/logger"
	"github.com/asianchinaboi/backendserver/internal/session"
	"github.com/asianchinaboi/backendserver/internal/wsclient"
	"github.com/gin-gonic/gin"
)

func Send(c *gin.Context) {
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

	msg.Content = strings.TrimSpace(msg.Content)
	//screw off html
	msg.Content = html.EscapeString(msg.Content) //prevents xss attacks
	msg.Created = time.Now().UnixMilli()
	logger.Debug.Printf("Message recieved %s\n", msg.Content)
	if len(msg.Content) == 0 {
		logger.Error.Println(errors.ErrNoMsgContent)
		c.JSON(http.StatusUnprocessableEntity, errors.Body{
			Error:  errors.ErrNoMsgContent.Error(),
			Status: errors.StatusNoMsgContent,
		})
		return
	}

	if len(msg.Content) > config.Config.Guild.MaxMsgLength {
		logger.Error.Println(errors.ErrMsgTooLong)
		c.JSON(http.StatusForbidden, errors.Body{
			Error:  errors.ErrMsgTooLong.Error(),
			Status: errors.StatusMsgTooLong,
		})
		return
	}

	if err := db.Db.QueryRow("INSERT INTO directmsgs (content, sender_id, dm_id, created) VALUES ($1, $2, $3, $4) RETURNING id", msg.Content, user.Id, dmId, msg.Created).Scan(&msg.MsgId); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	var authorBody events.User
	if err := db.Db.QueryRow("SELECT username FROM users WHERE id=$1", user.Id).Scan(&authorBody.Name); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	authorBody.UserId = user.Id
	authorBody.Icon = 0 //placeholder
	msg.Author = authorBody
	msg.DmId = intDmId

	var otherUser int

	if err := db.Db.QueryRow("SELECT user_id FROM userdirectmsgsguild WHERE dm_id = $1 AND user_id != $2 ", dmId, user.Id).Scan(&otherUser); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	wsclient.Pools.BroadcastClient(user.Id, wsclient.DataFrame{
		Op:    wsclient.TYPE_DISPATCH,
		Data:  msg,
		Event: events.CREATE_DM_MESSAGE,
	})
	wsclient.Pools.BroadcastClient(otherUser, wsclient.DataFrame{
		Op:    wsclient.TYPE_DISPATCH,
		Data:  msg,
		Event: events.CREATE_DM_MESSAGE,
	})
	c.Status(http.StatusNoContent)
}
