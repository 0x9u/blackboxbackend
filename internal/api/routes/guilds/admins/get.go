package admins

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

	var isDm bool
	if err := db.Db.QueryRow("SELECT EXISTS (SELECT 1 FROM guilds WHERE id = $1 AND dm = true)", guildId).Scan(&isDm); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	if isDm {
		errors.SendErrorResponse(c, errors.ErrGuildIsDm, errors.StatusGuildIsDm)
		return
	}

	admins := []events.User{}
	rows, err := db.Db.Query("SELECT ug.user_id, u.username, f.id FROM userguilds ug INNER JOIN users u ON u.id = ug.user_id LEFT JOIN files f ON f.user_id = u.id WHERE ug.guild_id=$1 AND admin = true", guildId)
	if err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	defer rows.Close()
	for rows.Next() {
		userAdmin := events.User{}
		var imageId sql.NullInt64
		rows.Scan(&userAdmin.UserId, &userAdmin.Name, imageId)
		if imageId.Valid {
			userAdmin.ImageId = imageId.Int64
		} else {
			userAdmin.ImageId = -1
		}
		admins = append(admins, userAdmin)
	}
	c.JSON(http.StatusOK, admins)
}
