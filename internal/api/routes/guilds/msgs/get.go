package msgs

import (
	"database/sql"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/asianchinaboi/backendserver/internal/api/middleware"
	"github.com/asianchinaboi/backendserver/internal/db"
	"github.com/asianchinaboi/backendserver/internal/errors"
	"github.com/asianchinaboi/backendserver/internal/events"
	"github.com/asianchinaboi/backendserver/internal/logger"
	"github.com/asianchinaboi/backendserver/internal/session"
	"github.com/gin-gonic/gin"
)

func Get(c *gin.Context) { //sends message history
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

	urlVars := c.Request.URL.Query()
	limit := urlVars.Get("limit")
	timestamp := urlVars.Get("time")

	if timestamp == "" {
		timestamp = fmt.Sprintf("%v", time.Now().Unix())
	}

	if limit == "" {
		limit = "50"
	}

	var inGuild bool

	if err := db.Db.QueryRow("SELECT EXISTS (SELECT * FROM userguilds WHERE guild_id=$1 AND user_id=$2 AND banned=false)", guildId, user.Id).Scan(&inGuild); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	if !inGuild {
		logger.Error.Println(errors.ErrNotInGuild)
		c.JSON(http.StatusForbidden, errors.Body{
			Error:  errors.ErrNotInGuild.Error(),
			Status: errors.StatusNotInGuild,
		})
		return
	}

	rows, err := db.Db.Query(
		`SELECT m.id, m.content, m.user_id, m.guild_id, m.created, m.modified, m.mentions_everyone, u.username, f.id
		FROM msgs m INNER JOIN users u 
		ON u.id = m.user_id LEFT JOIN files f
		ON f.user_id = u.id 
		WHERE m.created < $1 AND m.guild_id = $2 
		ORDER BY m.created DESC LIMIT $3`, //wtf? (i forgot what i did to make this work but it works anyways)
		timestamp, guildId, limit)
	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	defer rows.Close()
	messages := []events.Msg{}
	for rows.Next() {
		message := events.Msg{}
		var imageId sql.NullInt64
		var modified sql.NullInt64
		err := rows.Scan(&message.MsgId, &message.Content, &message.Author.UserId,
			&message.GuildId, &message.Created, &modified, &message.MentionsEveryone, &message.Author.Name, &imageId)
		if modified.Valid { //to make it show in json
			message.Modified = modified.Int64
		} else {
			message.Modified = -1
		}
		if imageId.Valid {
			message.Author.ImageId = imageId.Int64
		} else {
			message.Author.ImageId = -1
		}
		if err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}

		mentions, err := db.Db.Query(`SELECT mm.user_id, u.username FROM msgmentions mm INNER JOIN users u ON u.id = mm.user_id WHERE msg_id = $1`, message.MsgId)
		if err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}

		for mentions.Next() {
			var mentionUser events.User
			if err := mentions.Scan(&mentionUser.UserId, &mentionUser.Name); err != nil {
				logger.Error.Println(err)
				c.JSON(http.StatusInternalServerError, errors.Body{
					Error:  err.Error(),
					Status: errors.StatusInternalError,
				})
				return
			}
			message.Mentions = append(message.Mentions, mentionUser)
		}
		mentions.Close()

		attachments, err := db.Db.Query(`SELECT id, filename FROM files WHERE msg_id = $1`, message.MsgId)
		if err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}

		message.Attachments = []events.Attachment{}

		for attachments.Next() {
			var attachment events.Attachment
			if err := attachments.Scan(&attachment.Id, &attachment.Filename); err != nil {
				logger.Error.Println(err)
				c.JSON(http.StatusInternalServerError, errors.Body{
					Error:  err.Error(),
					Status: errors.StatusInternalError,
				})
				return
			}

			message.Attachments = append(message.Attachments, attachment)
		}
		attachments.Close()

		message.MsgSaved = true
		messages = append(messages, message)
	}
	c.JSON(http.StatusOK, messages)
}
