package bans

import (
	"database/sql"
	"net/http"
	"regexp"

	"github.com/asianchinaboi/backendserver/internal/api/middleware"
	"github.com/asianchinaboi/backendserver/internal/db"
	"github.com/asianchinaboi/backendserver/internal/errors"
	"github.com/asianchinaboi/backendserver/internal/events"
	"github.com/asianchinaboi/backendserver/internal/session"
	"github.com/gin-gonic/gin"
)

func Get(c *gin.Context) {
	user := c.MustGet(middleware.User).(*session.Session)
	if user == nil {
		errors.SendErrorResponse(c, errors.ErrSessionDidntPass, errors.StatusInternalError)
		return
	}

	guildId := c.Param("guildId")
	if match, err := regexp.MatchString("^[0-9]+$", guildId); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	} else if !match {
		errors.SendErrorResponse(c, errors.ErrRouteParamInvalid, errors.StatusRouteParamInvalid)
		return
	}

	var hasAuth bool
	var isDm bool

	if err := db.Db.QueryRow("SELECT EXISTS (SELECT 1 FROM userguilds WHERE guild_id=$1 AND user_id=$2 AND (owner=true OR admin=true)), EXISTS (SELECT 1 FROM guilds WHERE id = $1 AND dm = true)", guildId, user.Id).Scan(&hasAuth, &isDm); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}
	if isDm {
		errors.SendErrorResponse(c, errors.ErrGuildIsDm, errors.StatusGuildIsDm)
		return
	}

	if !hasAuth {
		errors.SendErrorResponse(c, errors.ErrNotGuildAuthorised, errors.StatusNotGuildAuthorised)
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
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}
	userlist := []events.Member{}
	if err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}
	for rows.Next() {
		var user events.Member
		var imageId sql.NullInt64
		if err := rows.Scan(&user.UserInfo.UserId, &user.UserInfo.Name, &imageId); err != nil {
			errors.SendErrorResponse(c, err, errors.StatusInternalError)
			return
		}
		if imageId.Valid {
			user.UserInfo.ImageId = imageId.Int64
		} else {
			user.UserInfo.ImageId = -1
		}
		userlist = append(userlist, user)
	}
	c.JSON(http.StatusOK, userlist)
}
