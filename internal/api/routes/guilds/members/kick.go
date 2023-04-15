package members

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

func Kick(c *gin.Context) {
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

	userId := c.Param("userId")
	if match, err := regexp.MatchString("^[0-9]+$", userId); err != nil {
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

	intGuildId, err := strconv.ParseInt(guildId, 10, 64)
	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	intUserId, err := strconv.ParseInt(userId, 10, 64)
	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	if intUserId == user.Id {
		logger.Error.Println(errors.ErrCantKickBanSelf)
		c.JSON(http.StatusForbidden, errors.Body{
			Error:  errors.ErrCantKickBanSelf.Error(),
			Status: errors.StatusCantKickBanSelf,
		})
		return
	}

	var isOwner bool
	var isAdmin bool
	var isUserAdmin bool
	var isUserInGuild bool
	var isDm bool

	if err := db.Db.QueryRow(`
	SELECT EXISTS (SELECT 1 FROM userguilds WHERE guild_id=$1 AND user_id=$2 AND owner=true), 
	EXISTS (SELECT 1 FROM guilds WHERE id = $1 AND dm = true),
	EXISTS (SELECT 1 FROM userguilds WHERE guild_id=$1 AND user_id=$2 AND admin=true),
	EXISTS (SELECT 1 FROM userguilds WHERE guild_id=$1 AND user_id=$3 AND admin=true),
	EXISTS (SELECT 1 FROM userguilds WHERE guild_id=$1 AND user_id=$3)
	`, guildId, user.Id, userId).Scan(&isOwner, &isDm, &isAdmin, &isUserAdmin, &isUserInGuild); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	if isDm {
		logger.Error.Println(errors.ErrGuildIsDm)
		c.JSON(http.StatusForbidden, errors.Body{
			Error:  errors.ErrGuildIsDm.Error(),
			Status: errors.StatusGuildIsDm,
		})
		return
	}

	if !isAdmin && !isOwner {
		logger.Error.Println(errors.ErrNotGuildAuthorised)
		c.JSON(http.StatusForbidden, errors.Body{
			Error:  errors.ErrNotGuildAuthorised.Error(),
			Status: errors.StatusNotGuildAuthorised,
		})
		return
	}

	if !isUserInGuild {
		logger.Error.Println(errors.ErrNotInGuild)
		c.JSON(http.StatusForbidden, errors.Body{
			Error:  errors.ErrNotInGuild.Error(),
			Status: errors.StatusNotInGuild,
		})
		return
	}

	if isUserAdmin && !isOwner {
		logger.Error.Println(errors.ErrNotAuthorised)
		c.JSON(http.StatusForbidden, errors.Body{
			Error:  errors.ErrNotAuthorised.Error(),
			Status: errors.StatusNotAuthorised,
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

	kickRes := wsclient.DataFrame{
		Op: wsclient.TYPE_DISPATCH,
		Data: events.Guild{
			GuildId: intGuildId,
		},
		Event: events.DELETE_GUILD,
	}
	guildRes := wsclient.DataFrame{
		Op: wsclient.TYPE_DISPATCH,
		Data: events.UserGuild{
			UserId:  intUserId,
			GuildId: intGuildId,
		},
		Event: events.REMOVE_USER_GUILDLIST,
	}
	wsclient.Pools.BroadcastClient(intUserId, kickRes)
	wsclient.Pools.RemoveUserFromGuildPool(intGuildId, intUserId)
	wsclient.Pools.BroadcastGuild(intGuildId, guildRes)
	c.Status(http.StatusNoContent)
}
