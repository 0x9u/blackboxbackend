package routes

import (
	"github.com/asianchinaboi/backendserver/internal/api/middleware"
	"github.com/asianchinaboi/backendserver/internal/api/routes/admin"
	"github.com/asianchinaboi/backendserver/internal/api/routes/files"
	"github.com/asianchinaboi/backendserver/internal/api/routes/guilds"
	"github.com/asianchinaboi/backendserver/internal/api/routes/static"
	"github.com/asianchinaboi/backendserver/internal/api/routes/status"
	"github.com/asianchinaboi/backendserver/internal/api/routes/users"
	"github.com/asianchinaboi/backendserver/internal/api/routes/ws"
	"github.com/gin-gonic/gin"
)

func PrepareRoutes(r *gin.Engine) {
	r.Use(middleware.CheckIP)
	static.Routes(r)
	apiRoute := r.Group("/api")
	apiRoute.Use(middleware.Cooldown)
	admin.Routes(apiRoute)
	status.Routes(apiRoute)
	guilds.Routes(apiRoute)
	users.Routes(apiRoute)
	ws.Routes(apiRoute)
	files.Routes(apiRoute)
}
