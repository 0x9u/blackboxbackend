package middleware

import (
	"net/http"

	"github.com/asianchinaboi/backendserver/internal/errors"
	"github.com/asianchinaboi/backendserver/internal/logger"
	"github.com/asianchinaboi/backendserver/internal/session"
	"github.com/gin-gonic/gin"
)

const User = "user"

func Auth(c *gin.Context) {
	token, ok := c.Request.Header["Authorization"]
	if !ok || len(token) == 0 {
		logger.Error.Println(errors.ErrAbsentToken)
		c.JSON(http.StatusUnauthorized, errors.Body{
			Error:  errors.ErrAbsentToken.Error(),
			Status: errors.StatusAbsentToken,
		})
		c.Abort()
		return
	}
	user, err := session.CheckToken(token[0])
	c.Set(User, user)
	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusForbidden, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusNotAuthorised,
		})
		c.Abort()
		return
	}
	c.Next()
}
