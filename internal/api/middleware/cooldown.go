package middleware

import (
	"github.com/asianchinaboi/backendserver/internal/cooldown"
	"github.com/asianchinaboi/backendserver/internal/errors"
	"github.com/gin-gonic/gin"
)

func Cooldown(c *gin.Context) {
	ip := c.ClientIP()
	canPass := cooldown.Manager.AddCount(ip)
	if !canPass {
		errors.SendErrorResponse(c, errors.ErrCooldownActive, errors.StatusCooldownActive)
		c.Abort()
		return
	}
	c.Next()
}
