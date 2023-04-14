package users

import (
	"database/sql"
	"net/http"

	"github.com/asianchinaboi/backendserver/internal/api/middleware"
	"github.com/asianchinaboi/backendserver/internal/db"
	"github.com/asianchinaboi/backendserver/internal/errors"
	"github.com/asianchinaboi/backendserver/internal/events"
	"github.com/asianchinaboi/backendserver/internal/logger"
	"github.com/asianchinaboi/backendserver/internal/session"
	"github.com/gin-gonic/gin"
)

type guildList struct {
	Guilds []events.Guild `json:"guilds"`
	Dms    []events.Dm    `json:"dms"`
}

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
	guildRows, err := db.Db.Query(
		/* long goofy aaaaa code*/
		`
		SELECT g.id, g.name, g.image_id, g.save_chat, (SELECT user_id FROM userguilds WHERE guild_id = u.guild_id AND owner = true) AS owner_id, un.msg_id AS last_read_msg_id, COUNT(m.id) filter (WHERE m.id > un.msg_id) AS unread_msgs, un.time
		FROM userguilds u 
		INNER JOIN guilds g ON g.id = u.guild_id 
		INNER JOIN unreadmsgs un ON un.guild_id = u.guild_id and un.user_id = u.user_id
		LEFT JOIN msgs m ON m.guild_id = un.guild_id 
		WHERE u.user_id=$1 AND u.banned = false
		GROUP BY g.id, g.name, g.image_id, owner_id, un.msg_id, un.time, u.*
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
	defer guildRows.Close()
	dmRows, err := db.Db.Query(
		`
		SELECT userg.guild_id, userg.receiver_id, users.username,
		unread.msg_id AS last_read_msg_id, COUNT(msgs.id) filter (WHERE msgs.id > unread.msg_id) AS unread_msgs,
		COUNT(mentions.msg_id) filter (WHERE mentions.user_id = $1 AND mentions.msg_id > unread.msg_id) + 
		COUNT(msgs.id) filter (WHERE msgs.mentions_everyone = true AND msgs.id > unread.msg_id) AS mentions, unread.time
		FROM userguilds userg
		INNER JOIN users ON users.id = userg.receiver_id
		INNER JOIN unreadmsgs unread ON unread.guild_id = msgs.guild_id AND unread.user_id = $1 
		LEFT JOIN msgs ON msgs.id = unread.msg_id 
		LEFT JOIN msgmentions mentions ON mentions.msg_id = msgs.id 
		WHERE userg.user_id=$1 AND userg.left_dm = false AND userg.receiver_id IS NOT NULL 
		GROUP BY userg.dm_id, userg.receiver_id, users.username, unread.time
		ORDER BY unread.time DESC
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
	defer dmRows.Close()
	guilds := []events.Guild{}
	for guildRows.Next() {
		var guild events.Guild
		var imageId sql.NullInt64
		guild.Unread = &events.UnreadMsg{}

		err = guildRows.Scan(&guild.GuildId, &guild.Name, &imageId, &guild.SaveChat, &guild.OwnerId, &guild.Unread.Id, &guild.Unread.Count, &guild.Unread.Time)
		if err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
		if imageId.Valid {
			guild.ImageId = imageId.Int64
		}
		guilds = append(guilds, guild)
	}
	dms := []events.Dm{}
	for dmRows.Next() {
		var dm events.Dm
		dm.Unread = events.UnreadMsg{}
		err = dmRows.Scan(&dm.DmId, &dm.UserInfo.UserId, &dm.UserInfo.Name, &dm.Unread.Id, &dm.Unread.Count, &dm.Unread.Mentions, &dm.Unread.Time)
		if err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
		dms = append(dms, dm)
	}
	c.JSON(http.StatusOK, guildList{
		Guilds: guilds,
		Dms:    dms,
	})
}
