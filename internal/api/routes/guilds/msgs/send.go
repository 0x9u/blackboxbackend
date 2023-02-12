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

// expects
// content : string
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

	guildId := c.Param("guildId")
	if match, err := regexp.MatchString("^[0-9]+$", guildId); err != nil {
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

	intGuildId, err := strconv.Atoi(guildId)
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

	//send msg to database
	//broadcast msg to all connections to websocket
	var inGuild bool
	if err := db.Db.QueryRow("SELECT EXISTS (SELECT * FROM userguilds WHERE guild_id=$1 AND user_id=$2 AND banned=false)", guildId, user.Id).Scan(&inGuild); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	if !inGuild {
		logger.Error.Println(errors.ErrNotInGuild)
		c.JSON(http.StatusForbidden, errors.Body{
			Error:  errors.ErrNotInGuild.Error(),
			Status: errors.StatusNotInGuild,
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

	//check if guild has chat messages save turned on
	var isChatSaveOn bool
	if err := db.Db.QueryRow("SELECT save_chat FROM guilds WHERE id=$1", guildId).Scan(&isChatSaveOn); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	msg.MsgId = 0 //just there to make it obvious

	if isChatSaveOn {
		if err := db.Db.QueryRow("INSERT INTO msgs (content, user_id, guild_id, created) VALUES ($1, $2, $3, $4) RETURNING id", msg.Content, user.Id, guildId, msg.Created).Scan(&msg.MsgId); err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
	}

	msg.MsgSaved = isChatSaveOn //false not saved | true saved

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
	wsclient.Pools.BroadcastGuild(intGuildId, wsclient.DataFrame{
		Op:    wsclient.TYPE_DISPATCH,
		Data:  msg,
		Event: events.CREATE_GUILD_MESSAGE,
	})
	c.Status(http.StatusNoContent)
}
