package users

import (
	"net/http"

	"github.com/asianchinaboi/backendserver/internal/api/middleware"
	"github.com/asianchinaboi/backendserver/internal/db"
	"github.com/asianchinaboi/backendserver/internal/errors"
	"github.com/asianchinaboi/backendserver/internal/events"
	"github.com/asianchinaboi/backendserver/internal/logger"
	"github.com/asianchinaboi/backendserver/internal/session"
	"github.com/gin-gonic/gin"
)

func getSelfGuilds(c *gin.Context) {
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
	logger.Info.Println("Getting guilds")
	rows, err := db.Db.Query(
		/* long goofy aaaaa code*/
		`
		SELECT g.id, g.name, g.icon, (SELECT user_id FROM userguilds WHERE guild_id = u.guild_id AND owner = true) AS owner_id, un.message_id AS last_read_msg_id, COUNT(m.id) filter (WHERE m.id > un.message_id) AS unread_msgs, un.time
		FROM userguilds u 
		INNER JOIN guilds g ON g.id = u.guild_id 
		INNER JOIN unreadmessages un ON un.guild_id = u.guild_id and un.user_id = u.user_id
		LEFT JOIN messages m ON m.guild_id = un.guild_id 
		WHERE u.user_id=$1 AND u.banned = false
		GROUP BY g.id, g.name, g.icon, owner_id, un.message_id, un.time, u.*
		ORDER BY u
		`,
		user.Id,
	)
	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	guilds := []events.Guild{}
	for rows.Next() {
		var guild events.Guild
		guild.Unread = &events.UnreadMsg{}
		err = rows.Scan(&guild.GuildId, &guild.Name, &guild.Icon, &guild.OwnerId, &guild.Unread.Id, &guild.Unread.Count, &guild.Unread.Time)
		if err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
		guilds = append(guilds, guild)
	}

	c.JSON(http.StatusOK, guilds)
}
