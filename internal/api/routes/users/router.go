package users

import (
	"github.com/asianchinaboi/backendserver/internal/api/middleware"
	"github.com/asianchinaboi/backendserver/internal/api/routes/users/blocked"
	"github.com/asianchinaboi/backendserver/internal/api/routes/users/directmsgs"
	"github.com/asianchinaboi/backendserver/internal/api/routes/users/friends"
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
	users.PATCH("/:userId/msgs/:msgId", msgs.Edit).Use(middleware.Auth)
	users.DELETE("/:userId/msgs/:msgId", msgs.Delete).Use(middleware.Auth)
	users.POST("/:userId/msgs/typing", msgs.Typing).Use(middleware.Auth)

	users.PUT("/dms/:userId", directmsgs.Create).Use(middleware.Auth)
	users.GET("/dms", directmsgs.Get).Use(middleware.Auth)
	users.DELETE("/dms/:userId", directmsgs.Delete).Use(middleware.Auth)

	users.PUT("/friends/:userId", friends.Create).Use(middleware.Auth)
	users.GET("/friends", friends.Get).Use(middleware.Auth)
	users.DELETE("/friends/:userId", friends.Delete).Use(middleware.Auth)
	users.POST("/friends/:userId/accept", friends.Accept).Use(middleware.Auth)

	users.PUT("/blocked/:userId", blocked.Create).Use(middleware.Auth)
	users.GET("/blocked", blocked.Get).Use(middleware.Auth)
	users.DELETE("/blocked/:userId", blocked.Delete).Use(middleware.Auth)

	users.GET("/guilds", getSelfGuilds).Use(middleware.Auth)
	users.DELETE("/guilds/:guildId", leaveGuild).Use(middleware.Auth)
}
