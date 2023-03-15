package middleware

import (
	"net/http"

	"github.com/asianchinaboi/backendserver/internal/cooldown"
	"github.com/asianchinaboi/backendserver/internal/errors"
	"github.com/asianchinaboi/backendserver/internal/logger"
	"github.com/gin-gonic/gin"
)

func Cooldown(c *gin.Context) {
	ip := c.Request.RemoteAddr
	canPass := cooldown.Manager.AddCount(ip)
	logger.Debug.Println(ip)
	if !canPass {
		logger.Error.Println(errors.ErrCooldownActive)
		c.JSON(http.StatusTooManyRequests, errors.Body{
			Error:  errors.ErrCooldownActive.Error(),
			Status: errors.StatusCooldownActive,
		})
		c.Abort()
		return
	}
	c.Next()
}
