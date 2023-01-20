package guilds

import (
	"net/http"
	"regexp"

	"github.com/asianchinaboi/backendserver/internal/api/middleware"
	"github.com/asianchinaboi/backendserver/internal/db"
	"github.com/asianchinaboi/backendserver/internal/errors"
	"github.com/asianchinaboi/backendserver/internal/logger"
	"github.com/asianchinaboi/backendserver/internal/session"
	"github.com/gin-gonic/gin"
)

func Delete(c *gin.Context) {
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
	if !user.Perms.Admin && !user.Perms.Guilds.Delete {
		logger.Error.Println(errors.ErrNotAuthorised)
		c.JSON(http.StatusForbidden, errors.Body{
			Error:  errors.ErrNotAuthorised.Error(),
			Status: errors.StatusNotAuthorised,
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

	if _, err := db.Db.Exec("DELETE FROM invites WHERE guild_id = $1", guildId); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	if _, err := db.Db.Exec("DELETE FROM userguilds WHERE guild_id = $1", guildId); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	if _, err := db.Db.Exec("DELETE FROM messages WHERE guild_id = $1", guildId); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	if _, err := db.Db.Exec("DELETE FROM unreadmessages WHERE guild_id = $1", guildId); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	if _, err := db.Db.Exec("DELETE FROM guilds WHERE id = $1", guildId); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	//TODO broadcast delete to guild pool
	c.Status(http.StatusNoContent)
}
