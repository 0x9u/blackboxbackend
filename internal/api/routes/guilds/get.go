package guilds

import (
	"database/sql"
	"net/http"
	"regexp"

	"github.com/asianchinaboi/backendserver/internal/api/middleware"
	"github.com/asianchinaboi/backendserver/internal/db"
	"github.com/asianchinaboi/backendserver/internal/errors"
	"github.com/asianchinaboi/backendserver/internal/events"
	"github.com/asianchinaboi/backendserver/internal/logger"
	"github.com/asianchinaboi/backendserver/internal/session"
	"github.com/gin-gonic/gin"
)

func getGuild(c *gin.Context) {
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

	var isInGuild bool
	var isDm bool

	if err := db.Db.QueryRow("SELECT EXISTS(SELECT 1 FROM userguilds WHERE user_id = $1 AND guild_id = $2 AND banned = false), EXISTS(SELECT 1 FROM guilds WHERE id = $2 AND dm = true)", user.Id, guildId).Scan(&isInGuild); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
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

	if isDm {
		logger.Error.Println(errors.ErrGuildIsDm)
		c.JSON(http.StatusForbidden, errors.Body{
			Error:  errors.ErrGuildIsDm.Error(),
			Status: errors.StatusGuildIsDm,
		})
		return
	}

	query := `
		SELECT SELECT g.id, g.name, f.id, g.save_chat, 
		(SELECT user_id FROM userguilds WHERE guild_id = $1 AND owner = true) AS owner_id, 
		un.msg_id AS last_read_msg_id, COUNT(m.id) filter (WHERE m.created > un.time) AS unread_msgs,
		un.time, COUNT(mm.msg_id) filter (WHERE mm.user_id = $2 AND m.created > un.time) +
		 COUNT(m.id) filter (WHERE m.mentions_everyone = true AND m.created > un.time) AS mentions FROM guilds g
		INNER JOIN unreadmsgs un ON un.guild_id = g.id AND un.user_id = $2
		LEFT JOIN msgs m ON m.guild_id = g.id 
		LEFT JOIN msgmentions mm ON m.id = mm.msg_id
		LEFT JOIN files f ON f.guild_id = g.id 
		WHERE g.id = $1 
		GROUP BY g.id, g.name, f.id, owner_id, un.msg_id, un.time
	`
	var guild events.Guild
	var imageId sql.NullInt64
	if err := db.Db.QueryRow(query, guildId, user.Id).Scan(&guild.GuildId,
		&guild.Name, &imageId,
		&guild.SaveChat, &guild.OwnerId,
		&guild.Unread.Id, &guild.Unread.Count,
		&guild.Unread.Time); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	if imageId.Valid {
		guild.ImageId = imageId.Int64
	} else {
		guild.ImageId = -1
	}
	c.JSON(
		http.StatusOK,
		guild,
	)
}
