package users

import (
	"github.com/asianchinaboi/backendserver/internal/api/middleware"
	"github.com/asianchinaboi/backendserver/internal/api/routes/users/msgs"
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

	users.GET("/:userId", getUserInfo).Use(middleware.Auth)

	users.POST("/:userId/msgs", msgs.Send).Use(middleware.Auth)
	users.GET("/:userId/msgs", msgs.Get).Use(middleware.Auth)
	users.DELETE("/:userId/msgs/:msgId", msgs.Delete).Use(middleware.Auth)

	users.POST("/dms", msgs.CreateDM).Use(middleware.Auth)
	users.GET("/dms", msgs.GetDM).Use(middleware.Auth)

	users.GET("/guilds", getSelfGuilds).Use(middleware.Auth)
	users.DELETE("/guilds/:guildId", leaveGuild).Use(middleware.Auth)
}
