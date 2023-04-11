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

	dmId := c.Param("dmId")
	if match, err := regexp.MatchString("^[0-9]+$", dmId); err != nil {
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

	var openDM bool

	if err := db.Db.QueryRow("SELECT EXISTS (SELECT 1 FROM userdirectmsgsguild WHERE dm_id = $1 AND user_id = $2) ", dmId, user.Id).Scan(&openDM); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	if !openDM {
		logger.Error.Println(errors.ErrDMNotOpened)
		c.JSON(http.StatusForbidden, errors.Body{
			Error:  errors.ErrDMNotOpened.Error(),
			Status: errors.StatusDMNotOpened,
		})
		return
	}

	logger.Debug.Printf("limit: %v, timestamp %v\n", limit, timestamp)
	rows, err := db.Db.Query(
		`SELECT m.*, u.username, u.image_id
	FROM directmsgs m INNER JOIN users u 
	ON u.user_id = m.user_id 
	WHERE created < $1 AND dm_id = $2
	ORDER BY created DESC LIMIT $3`,
		timestamp, dmId, limit)
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
			&message.DmId, &message.Created, &message.Modified, &message.MentionsEveryone, &message.Author.Name, &message.Author.ImageId)
		if message.Modified == 0 { //to make it show in json
			message.Modified = -1
		}
		if err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
		mentions, err := db.Db.Query(`SELECT mm.user_id, u.username FROM directmsgmentions mm INNER JOIN users u ON u.id = mm.user_id WHERE directmsg_id = $1`, message.MsgId)
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

		message.MsgSaved = true
		messages = append(messages, message)
	}
	c.JSON(http.StatusOK, messages)
}
