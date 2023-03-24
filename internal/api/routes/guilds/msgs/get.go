package msgs

import (
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

	logger.Debug.Printf("limit: %v, timestamp %v\n", limit, timestamp)
	rows, err := db.Db.Query(
		`SELECT m.*, u.username
		FROM msgs m INNER JOIN users u 
		ON u.id = m.user_id 
		WHERE created < $1 AND guild_id = $2 
		ORDER BY created DESC LIMIT $3`, //wtf? (i forgot what i did to make this work but it works anyways)
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
		err := rows.Scan(&message.MsgId, &message.Content, &message.Author.UserId,
			&message.GuildId, &message.Created, &message.Modified, &message.Author.Name)
		if message.Modified == 0 { //to make it show in json
			message.Modified = -1
		}
		message.Author.Icon = 0 //placeholder
		if err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
		message.MsgSaved = true
		messages = append(messages, message)
	}
	c.JSON(http.StatusOK, messages)
}
