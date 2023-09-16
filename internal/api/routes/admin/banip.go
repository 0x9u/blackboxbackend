package admin

import (
	"net/http"

	"github.com/asianchinaboi/backendserver/internal/api/middleware"
	"github.com/asianchinaboi/backendserver/internal/db"
	"github.com/asianchinaboi/backendserver/internal/errors"
	"github.com/asianchinaboi/backendserver/internal/session"
	"github.com/gin-gonic/gin"
)

type banIPBody struct {
	IP string `json:"ip"`
}

func banIP(c *gin.Context) {
	user := c.MustGet(middleware.User).(*session.Session)
	if user == nil {
		errors.SendErrorResponse(c, errors.ErrSessionDidntPass, errors.StatusInternalError)
		return
	}
	if !user.Perms.Admin && !user.Perms.BanIP {
		errors.SendErrorResponse(c, errors.ErrNotAuthorised, errors.StatusNotAuthorised)
		return
	}
	var body banIPBody
	if err := c.ShouldBindJSON(&body); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}
	if _, err := db.Db.Exec("INSERT INTO bannedips (ip) VALUES ($1) ON CONFLICT DO NOTHING", body.IP); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}
	c.Status(http.StatusNoContent)
}
