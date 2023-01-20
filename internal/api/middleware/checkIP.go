package middleware

import (
	"net/http"

	"github.com/asianchinaboi/backendserver/internal/db"
	"github.com/asianchinaboi/backendserver/internal/errors"
	"github.com/asianchinaboi/backendserver/internal/logger"
	"github.com/gin-gonic/gin"
)

func CheckIP(c *gin.Context) {
	var isBanned bool
	ip := c.Request.RemoteAddr
	if err := db.Db.QueryRow("SELECT EXISTS (SELECT 1 FROM bannedips WHERE ip = $1)", ip).Scan(&isBanned); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		c.Abort()
		return
	}
	if isBanned {
		logger.Error.Println(errors.ErrIpBanned)
		c.JSON(http.StatusForbidden, errors.Body{
			Error:  errors.ErrIpBanned.Error(),
			Status: errors.StatusIpBanned,
		})
		return
	}
	c.Next()
}
