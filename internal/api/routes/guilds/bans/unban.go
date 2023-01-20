package bans

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

func Unban(c *gin.Context) {
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
		logger.Error.Println(errors.ErrRouteParamNotInt)
		c.JSON(http.StatusBadRequest, errors.Body{
			Error:  errors.ErrRouteParamNotInt.Error(),
			Status: errors.StatusRouteParamNotInt,
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
		logger.Error.Println(errors.ErrRouteParamNotInt)
		c.JSON(http.StatusBadRequest, errors.Body{
			Error:  errors.ErrRouteParamNotInt.Error(),
			Status: errors.StatusRouteParamNotInt,
		})
		return
	}

	var isOwner bool
	var isBanned bool

	if err := db.Db.QueryRow("SELECT EXISTS (SELECT 1 FROM userguilds WHERE guild_id=$1 AND user_id=$2 AND owner=true), EXISTS (SELECT 1 FROM userguilds WHERE guild_id=$1 AND user_id=$3 AND banned = true)", guildId, user.Id, userId).Scan(&isOwner, &isBanned); err != nil {
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
	if !isBanned {
		logger.Error.Println(errors.ErrUserNotBanned)
		c.JSON(http.StatusForbidden, errors.Body{
			Error:  errors.ErrUserNotBanned.Error(),
			Status: errors.StatusUserNotBanned,
		})
		return
	}

	if _, err := db.Db.Exec("DELETE FROM userguilds WHERE guild_id=$1 AND user_id=$2", guildId, userId); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
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
	intUserId, err := strconv.Atoi(userId)
	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	res := wsclient.DataFrame{
		Op: wsclient.TYPE_DISPATCH,
		Data: events.UserGuild{
			GuildId: intGuildId,
			UserId:  intUserId,
		},
		Event: events.REMOVE_USER_BANLIST,
	}
	wsclient.Pools.BroadcastClient(user.Id, res)
	c.Status(http.StatusNoContent)
}
