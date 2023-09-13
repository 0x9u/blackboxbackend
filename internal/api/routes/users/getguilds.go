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
		SELECT g.id, g.name, f.id, g.save_chat, (SELECT user_id FROM userguilds WHERE guild_id = u.guild_id AND owner = true) AS owner_id, 
		un.msg_id AS last_read_msg_id, 
		COUNT(m.id) filter (WHERE m.created > un.time) AS unread_msgs, 
		COUNT(mentions.msg_id) filter (WHERE mentions.user_id = $1 AND m.created > un.time) + 
		COUNT(m.id) filter (WHERE m.mentions_everyone = true AND m.created > un.time) AS mentions, 
		un.time 
		FROM userguilds u 
		INNER JOIN guilds g ON g.id = u.guild_id 
		INNER JOIN unreadmsgs un ON un.guild_id = u.guild_id AND un.user_id = u.user_id
		LEFT JOIN msgs m ON m.guild_id = un.guild_id
		LEFT JOIN files f ON f.guild_id = g.id 
		LEFT JOIN msgmentions mentions ON mentions.msg_id = m.id  
		WHERE u.user_id=$1 AND u.banned = false AND g.dm = false 
		GROUP BY g.id, g.name, f.id, owner_id, un.msg_id, un.time, u.*
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
		SELECT userg.guild_id, userg.receiver_id, users.username, files.id, 
		unread.msg_id AS last_read_msg_id, COUNT(msgs.id) filter (WHERE msgs.created > unread.time) AS unread_msgs,
		COUNT(mentions.msg_id) filter (WHERE mentions.user_id = $1 AND msgs.created > unread.time) + 
		COUNT(msgs.id) filter (WHERE msgs.mentions_everyone = true AND msgs.created > unread.time) AS mentions, unread.time
		FROM userguilds userg 
		INNER JOIN users ON users.id = userg.receiver_id 
		INNER JOIN unreadmsgs unread ON unread.guild_id = userg.guild_id AND unread.user_id = $1 
		LEFT JOIN msgs ON msgs.guild_id = userg.guild_id 
		LEFT JOIN msgmentions mentions ON mentions.msg_id = msgs.id
		LEFT JOIN files ON files.user_id = users.id 
		WHERE userg.user_id=$1 AND userg.left_dm = false AND userg.receiver_id IS NOT NULL 
		GROUP BY userg.guild_id, userg.receiver_id, users.username, files.id, unread.msg_id, unread.time
		ORDER BY unread.time DESC
		`, //holy shit it works
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

		err = guildRows.Scan(&guild.GuildId, &guild.Name, &imageId, &guild.SaveChat, &guild.OwnerId, &guild.Unread.MsgId, &guild.Unread.Count, &guild.Unread.Mentions, &guild.Unread.Time)
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
		} else {
			guild.ImageId = -1
		}
		guilds = append(guilds, guild)
	}
	dms := []events.Dm{}
	for dmRows.Next() {
		var dm events.Dm
		var imageId sql.NullInt64
		dm.Unread = events.UnreadMsg{}
		err = dmRows.Scan(&dm.DmId, &dm.UserInfo.UserId, &dm.UserInfo.Name, &imageId, &dm.Unread.MsgId, &dm.Unread.Count, &dm.Unread.Mentions, &dm.Unread.Time)
		if err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
		if imageId.Valid {
			dm.UserInfo.ImageId = imageId.Int64
		} else {
			dm.UserInfo.ImageId = -1
		}
		logger.Debug.Println(dm)
		dms = append(dms, dm)
	}
	c.JSON(http.StatusOK, guildList{
		Guilds: guilds,
		Dms:    dms,
	})
}
