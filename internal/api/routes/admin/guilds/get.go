package guilds

import (
	"database/sql"
	"net/http"
	"regexp"
	"strconv"

	"github.com/asianchinaboi/backendserver/internal/api/middleware"
	"github.com/asianchinaboi/backendserver/internal/db"
	"github.com/asianchinaboi/backendserver/internal/errors"
	"github.com/asianchinaboi/backendserver/internal/events"
	"github.com/asianchinaboi/backendserver/internal/logger"
	"github.com/asianchinaboi/backendserver/internal/session"
	"github.com/gin-gonic/gin"
)

func Get(c *gin.Context) {
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

	if !user.Perms.Admin && !user.Perms.Guilds.Get {
		logger.Error.Println(errors.ErrNotAuthorised)
		c.JSON(http.StatusForbidden, errors.Body{
			Error:  errors.ErrNotAuthorised.Error(),
			Status: errors.StatusNotAuthorised,
		})
		return
	}
	queryParms := c.Request.URL.Query()
	//GET PAGE NUM
	page := queryParms.Get("page")
	//GET LIMIT
	limit := queryParms.Get("limit")
	if match, err := regexp.MatchString(`^[0-9]+$`, page); !match {
		page = "0"
	} else if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	if match, err := regexp.MatchString(`^[0-9]+$`, limit); !match {
		limit = "0"
	} else if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	intPage, err := strconv.ParseInt(page, 10, 64)
	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	intLimit, err := strconv.ParseInt(limit, 10, 64)
	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	offset := intPage * intLimit
	var nullIntLimit sql.NullInt64
	if limit == "0" {
		nullIntLimit.Valid = false
	} else {
		nullIntLimit.Int64 = intLimit
		nullIntLimit.Valid = true
	}
	rows, err := db.Db.Query("SELECT guilds.*, files.id FROM guilds LEFT JOIN files ON files.guild_id = guilds.id LIMIT $1 OFFSET $2", nullIntLimit, offset)
	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	defer rows.Close()
	var guilds []events.Guild
	for rows.Next() {
		var guild events.Guild
		var imageId sql.NullInt64
		rows.Scan(&guild.GuildId, &guild.Name, &guild.SaveChat, &guild.Dm, &guild.ImageId)
		if imageId.Valid {
			guild.ImageId = imageId.Int64
		} else {
			guild.ImageId = -1
		}
		guilds = append(guilds, guild)
	}
	c.JSON(http.StatusOK, guilds)
}
