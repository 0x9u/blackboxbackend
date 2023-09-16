package middleware

import (
	"github.com/asianchinaboi/backendserver/internal/db"
	"github.com/asianchinaboi/backendserver/internal/errors"
	"github.com/gin-gonic/gin"
)

func CheckIP(c *gin.Context) {
	var isBanned bool
	ip := c.Request.RemoteAddr
	if err := db.Db.QueryRow("SELECT EXISTS (SELECT 1 FROM bannedips WHERE ip = $1)", ip).Scan(&isBanned); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		c.Abort()
		return
	}
	if isBanned {
		errors.SendErrorResponse(c, errors.ErrIpBanned, errors.StatusIpBanned)
		c.Abort()
		return
	}
	c.Next()
}
