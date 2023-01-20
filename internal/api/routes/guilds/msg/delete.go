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
		logger.Error.Println(errors.ErrRouteParamNotInt)
		c.JSON(http.StatusBadRequest, errors.Body{
			Error:  errors.ErrRouteParamNotInt.Error(),
			Status: errors.StatusRouteParamNotInt,
		})
		return
	}
	intMsgId, err := strconv.Atoi(msgId)
	var isRequestId bool
	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	isRequestId = err != nil

	guildId := c.Param("guildId")
	if match, err := regexp.MatchString("^[0-9]+$", guildId); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	} else if !match {
		logger.Error.Println(errors.ErrRouteParamNotInt)
		c.JSON(http.StatusBadRequest, errors.Body{
			Error:  errors.ErrRouteParamNotInt.Error(),
			Status: errors.StatusRouteParamNotInt,
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

	var isOwner bool
	var isAuthor bool
	if err := db.Db.QueryRow("SELECT EXISTS (SELECT 1 FROM userguilds WHERE guild_id=$1 AND user_id=$2 AND owner=true)", guildId, user.Id).Scan(&isOwner); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	if !isOwner {
		logger.Error.Println(errors.ErrNotGuildOwner)
		c.JSON(http.StatusForbidden, errors.Body{
			Error:  errors.ErrNotGuildOwner.Error(),
			Status: errors.StatusNotGuildOwner,
		})
		return
	}
	if err := db.Db.QueryRow("SELECT EXISTS (SELECT 1 FROM messages WHERE id = $1 AND user_id = $2)", msgId, user.Id).Scan(&isAuthor); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	if !isOwner && !isAuthor {
		logger.Error.Println(errors.ErrNotGuildOwner)
		c.JSON(http.StatusForbidden, errors.Body{
			Error:  errors.ErrNotGuildOwner.Error(),
			Status: errors.StatusNotGuildOwner,
		})
		return
	}

	//theortically if the user did not have chat saved and ran this request it would still work
	if isRequestId {
		var isChatSaved bool
		if err := db.Db.QueryRow("SELECT save_chat from guilds where id = $1", guildId).Scan(&isChatSaved); err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
		if !isChatSaved { //if save chat is on
			logger.Error.Println(errors.ErrGuildSaveChatOn)
			c.JSON(http.StatusForbidden, errors.Body{
				Error:  errors.ErrGuildSaveChatOn.Error(),
				Status: errors.StatusGuildSaveChatOn,
			})
			return
		}
	} else {
		if _, err = db.Db.Exec("DELETE FROM messages where id = $1 AND guild_id = $2 AND user_id = $3", msgId, guildId, user.Id); err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
	}

	var requestId string
	if isRequestId {
		requestId = msgId
	} else {
		requestId = "" //there for readabilty
	}

	res := wsclient.DataFrame{
		Op: wsclient.TYPE_DISPATCH,
		Data: events.Msg{
			MsgId:     intMsgId,
			GuildId:   intGuildId,
			RequestId: requestId,
		},
		Event: events.DELETE_MESSAGE,
	}
	wsclient.Pools.BroadcastGuild(intGuildId, res)
	c.Status(http.StatusNoContent)
}

func Clear(c *gin.Context) {
	user := c.MustGet(middleware.User).(*session.Session)

	guildId := c.Param("guildId")
	if match, err := regexp.MatchString("^[0-9]+$", guildId); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	} else if !match {
		logger.Error.Println(errors.ErrRouteParamNotInt)
		c.JSON(http.StatusBadRequest, errors.Body{
			Error:  errors.ErrRouteParamNotInt.Error(),
			Status: errors.StatusRouteParamNotInt,
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
	var isOwner bool
	var isChatSaved bool
	if err := db.Db.QueryRow("SELECT EXISTS (SELECT 1 FROM userguilds WHERE guild_id=$1 AND user_id=$2 AND owner=true)", guildId, user.Id).Scan(&isOwner); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	if !isOwner {
		logger.Error.Println(errors.ErrNotGuildOwner)
		c.JSON(http.StatusForbidden, errors.Body{
			Error:  errors.ErrNotGuildOwner.Error(),
			Status: errors.StatusNotGuildOwner,
		})
		return
	}

	if err := db.Db.QueryRow("SELECT save_chat from guilds where id = $1", guildId).Scan(&isChatSaved); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	if isChatSaved {
		if _, err := db.Db.Exec("DELETE FROM messages WHERE guild_id = $1", guildId); err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}

	}
	res := wsclient.DataFrame{
		Op: wsclient.TYPE_DISPATCH,
		Data: events.Guild{
			GuildId: intGuildId,
		},
		Event: events.CLEAR_GUILD_MESSAGES,
	}
	wsclient.Pools.BroadcastGuild(intGuildId, res)
	c.Status(http.StatusNoContent)
}
