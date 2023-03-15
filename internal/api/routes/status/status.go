package status

import (
	"net/http"

	"github.com/asianchinaboi/backendserver/internal/db"
	"github.com/asianchinaboi/backendserver/internal/errors"
	"github.com/asianchinaboi/backendserver/internal/logger"
	"github.com/asianchinaboi/backendserver/internal/wsclient"
	"github.com/gin-gonic/gin"
)

type statusInfo struct {
	ClientNumber    int `json:"clientNumber"`
	GuildNumber     int `json:"guildNumber"`
	MsgNumber       int `json:"msgNumber"`
	GuildPoolNumber int `json:"guildPoolNumber"`
}

func ShowStatus(c *gin.Context) { //debugging
	row := db.Db.QueryRow("SELECT COUNT(*) FROM messages")
	if err := row.Err(); err != nil {
		logger.Info.Println(err)
		c.JSON(http.StatusInternalServerError,
			errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
		return
	}
	var msgNumber int
	row.Scan(&msgNumber)
	row = db.Db.QueryRow("SELECT COUNT(*) FROM guilds")
	if err := row.Err(); err != nil {
		logger.Info.Println(err)
		c.JSON(http.StatusInternalServerError,
			errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
		return
	}
	var guildNumber int
	row.Scan(&guildNumber)

	guildPoolNumber := wsclient.Pools.GetLengthGuilds()

	status := statusInfo{
		ClientNumber:    wsclient.Pools.GetLengthClients(),
		GuildNumber:     guildNumber,
		MsgNumber:       msgNumber,
		GuildPoolNumber: guildPoolNumber,
	}
	c.JSON(http.StatusOK, status)
}
