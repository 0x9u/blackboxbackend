package bans

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

	var hasAuth bool
	var isDm bool

	if err := db.Db.QueryRow("SELECT EXISTS (SELECT 1 FROM userguilds WHERE guild_id=$1 AND user_id=$2 AND (owner=true OR admin=true)), EXISTS (SELECT 1 FROM guilds WHERE id = $1 AND dm = true)", guildId, user.Id).Scan(&hasAuth, &isDm); err != nil {
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

	if !hasAuth {
		logger.Error.Println(errors.ErrNotGuildAuthorised)
		c.JSON(http.StatusForbidden, errors.Body{
			Error:  errors.ErrNotGuildAuthorised.Error(),
			Status: errors.StatusNotGuildAuthorised,
		})
		return
	}

	rows, err := db.Db.Query(
		`
		SELECT u.id, u.username, f.id
		FROM userguilds g INNER JOIN users u ON u.id = g.user_id 
		LEFT JOIN files f ON f.user_id = u.id 
		WHERE g.banned = true AND g.guild_id = $1`,
		guildId,
	)
	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	userlist := []events.Member{}
	intGuildId, err := strconv.Atoi(guildId)
	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	for rows.Next() {
		var user events.Member
		var imageId sql.NullInt64
		if err := rows.Scan(&user.UserInfo.UserId, &user.UserInfo.Name, &imageId); err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
		if imageId.Valid {
			user.UserInfo.ImageId = imageId.Int64
		} else {
			user.UserInfo.ImageId = -1
		}
		user.GuildId = intGuildId
		userlist = append(userlist, user)
	}
	c.JSON(http.StatusOK, userlist)
}
