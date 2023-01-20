package users

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

func leaveGuild(c *gin.Context) {
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

	var isOwner bool
	var isInGuild bool

	if err := db.Db.QueryRow(`
	SELECT EXISTS (SELECT 1 FROM userguilds WHERE guild_id=$1 AND user_id=$2 AND owner = true),
	EXISTS(SELECT 1 FROM userguilds WHERE guild_id=$1 AND user_id=$2 AND banned = false)
	`, guildId, user.Id).Scan(&isOwner, &isInGuild); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	if isOwner {
		logger.Error.Println(errors.ErrCantLeaveOwnGuild)
		c.JSON(http.StatusForbidden, errors.Body{
			Error:  errors.ErrCantLeaveOwnGuild.Error(),
			Status: errors.StatusCantLeaveOwnGuild,
		})
		return
	}
	if !isInGuild {
		logger.Error.Println(errors.ErrNotInGuild)
		c.JSON(http.StatusForbidden, errors.Body{
			Error:  errors.ErrNotInGuild.Error(),
			Status: errors.StatusNotInGuild,
		})
		return
	}
	if _, err := db.Db.Exec("DELETE FROM userguilds WHERE guild_id=$1 AND user_id=$2", guildId, user.Id); err != nil {
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

	res := wsclient.DataFrame{
		Op: wsclient.TYPE_DISPATCH,
		Data: events.Guild{
			GuildId: intGuildId,
		},
		Event: events.DELETE_GUILD,
	}
	guildRes := wsclient.DataFrame{
		Op: wsclient.TYPE_DISPATCH,
		Data: events.UserGuild{
			GuildId: intGuildId,
			UserId:  user.Id,
		},
		Event: events.REMOVE_USER_GUILDLIST,
	}
	wsclient.Pools.BroadcastClient(user.Id, res)
	wsclient.Pools.RemoveUserFromGuildPool(intGuildId, user.Id)
	wsclient.Pools.BroadcastGuild(intGuildId, guildRes)
	c.Status(http.StatusNoContent)
}
