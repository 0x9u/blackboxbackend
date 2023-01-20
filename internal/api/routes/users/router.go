package users

import (
	"github.com/asianchinaboi/backendserver/internal/api/middleware"
	"github.com/gin-gonic/gin"
)

func Routes(r *gin.RouterGroup) {
	users := r.Group("/users")
	users.POST("/", userCreate)
	users.DELETE("/", userDelete)

	users.DELETE("/msgs", clearUserMsg)

	users.POST("/auth", userAuth)

	users.PATCH("/@me", editSelf).Use(middleware.Auth)
	users.GET("/@me", getSelfInfo).Use(middleware.Auth)

	users.GET("/guilds", getSelfGuilds).Use(middleware.Auth)
	users.DELETE("/guilds/:guildId", leaveGuild).Use(middleware.Auth)
}
