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
	FileNumber      int `json:"fileNumber"`
	GuildPoolNumber int `json:"guildPoolNumber"`
}

func ShowStatus(c *gin.Context) { //debugging
	var msgNumber int
	if err := db.Db.QueryRow("SELECT COUNT(*) FROM msgs").Scan(&msgNumber); err != nil {
		logger.Info.Println(err)
		c.JSON(http.StatusInternalServerError,
			errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
		return
	}

	var guildNumber int
	if err := db.Db.QueryRow("SELECT COUNT(*) FROM guilds").Scan(&guildNumber); err != nil {
		logger.Info.Println(err)
		c.JSON(http.StatusInternalServerError,
			errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
		return
	}

	var fileNumber int
	if err := db.Db.QueryRow("SELECT COUNT(*) FROM files").Scan(&fileNumber); err != nil {
		logger.Info.Println(err)
		c.JSON(http.StatusInternalServerError,
			errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
		return
	}

	guildPoolNumber := wsclient.Pools.GetLengthGuilds()

	status := statusInfo{
		ClientNumber:    wsclient.Pools.GetLengthClients(),
		GuildNumber:     guildNumber,
		MsgNumber:       msgNumber,
		GuildPoolNumber: guildPoolNumber,
	}
	c.JSON(http.StatusOK, status)
}
