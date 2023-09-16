package middleware

import (
	"github.com/asianchinaboi/backendserver/internal/errors"
	"github.com/asianchinaboi/backendserver/internal/session"
	"github.com/gin-gonic/gin"
)

const User = "user"

func Auth(c *gin.Context) {
	token, ok := c.Request.Header["Authorization"]
	if !ok || len(token) == 0 {
		errors.SendErrorResponse(c, errors.ErrAbsentToken, errors.StatusAbsentToken)
		c.Abort()
		return
	}
	user, err := session.CheckToken(token[0])
	c.Set(User, user)
	if err != nil {
		errors.SendErrorResponse(c, err, errors.StatusNotAuthorised)
		c.Abort()
		return
	}
	c.Next()
}
