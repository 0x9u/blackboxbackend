package guilds

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

type editGuildBody struct {
	SaveChat *bool   `json:"saveChat"`
	Name     *string `json:"name"`
	Icon     *int    `json:"icon"`
}

func editGuild(c *gin.Context) {
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

	var newSettings editGuildBody
	if err := c.ShouldBindJSON(&newSettings); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusBadRequest, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusBadJSON,
		})
		return
	}

	if newSettings.SaveChat == nil && newSettings.Icon == nil && newSettings.Name == nil {
		logger.Error.Println(errors.ErrAllFieldsEmpty)
		c.JSON(http.StatusBadRequest, errors.Body{
			Error:  errors.ErrAllFieldsEmpty.Error(),
			Status: errors.StatusAllFieldsEmpty,
		})
		return
	}

	var exists bool
	var isOwner bool
	if err := db.Db.QueryRow("SELECT EXISTS(SELECT * FROM guilds WHERE id = $1), EXISTS(SELECT * FROM userguilds WHERE user_id=$2 and guild_id=$1 and owner = true)", guildId, user.Id).Scan(&exists, &isOwner); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	if !exists {
		c.JSON(http.StatusNotFound, errors.Body{
			Error:  errors.ErrGuildNotExist.Error(),
			Status: errors.StatusGuildNotExist,
		})
		return
	}
	if !isOwner {
		c.JSON(http.StatusForbidden, errors.Body{
			Error:  errors.ErrNotGuildOwner.Error(),
			Status: errors.StatusNotGuildOwner,
		})
		return
	}

	bodyRes := events.Guild{}
	intGuildId, err := strconv.Atoi(guildId)
	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	bodyRes.GuildId = intGuildId

	bodySettingsRes := events.Guild{}

	bodySettingsRes.GuildId = intGuildId

	if newSettings.SaveChat != nil {

		if _, err = db.Db.Exec("UPDATE guilds SET save_chat=$1 WHERE id=$2", *newSettings.SaveChat, guildId); err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
		bodySettingsRes.SaveChat = newSettings.SaveChat
	}
	if newSettings.Name != nil {
		if valid, err := events.ValidateGuildName(*newSettings.Name); err != nil {
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
		} else if !valid {
			c.JSON(http.StatusUnprocessableEntity, errors.Body{
				Error:  errors.ErrInvalidGuildName.Error(),
				Status: errors.StatusInvalidGuildName,
			})
			return
		}

		if _, err = db.Db.Exec("UPDATE guilds SET name=$1 WHERE id=$2", *newSettings.Name, guildId); err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
		bodyRes.Name = *newSettings.Name
	}
	if newSettings.Icon != nil {
		if _, err = db.Db.Exec("UPDATE guilds SET icon=$1 WHERE id=$2", *newSettings.Icon, guildId); err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
		bodyRes.Icon = *newSettings.Icon
	}

	guildRes := wsclient.DataFrame{
		Op:    wsclient.TYPE_DISPATCH,
		Data:  bodyRes,
		Event: events.UPDATE_GUILD,
	}
	res := wsclient.DataFrame{
		Op:    wsclient.TYPE_DISPATCH,
		Data:  bodySettingsRes,
		Event: events.UPDATE_GUILD_SETTINGS,
	}
	wsclient.Pools.BroadcastGuild(intGuildId, guildRes)
	logger.Debug.Println(user.Id)
	wsclient.Pools.BroadcastClient(user.Id, res)

	c.Status(http.StatusNoContent)
}
