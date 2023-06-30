package users

import (
	"github.com/asianchinaboi/backendserver/internal/api/middleware"
	"github.com/asianchinaboi/backendserver/internal/api/routes/users/blocked"
	"github.com/asianchinaboi/backendserver/internal/api/routes/users/directmsgs"
	"github.com/asianchinaboi/backendserver/internal/api/routes/users/friends"
	"github.com/asianchinaboi/backendserver/internal/api/routes/users/requests"
	"github.com/gin-gonic/gin"
)

func Routes(r *gin.RouterGroup) {
	users := r.Group("/users")
	users.POST("/", userCreate)
	users.GET("/:userId", middleware.Auth, getUserInfo)

	users.POST("/auth", userAuth)

	self := users.Group("/@me").Use(middleware.Auth)

	self.PATCH("/", editSelf)
	self.DELETE("/", userDelete)
	self.GET("/", getSelfInfo)

	self.POST("/dms", directmsgs.Create)
	self.DELETE("/dms/:dmId", directmsgs.Delete)

	self.PUT("/friends", friends.CreateByName)
	self.PUT("/friends/:userId", friends.Create)
	self.GET("/friends", friends.Get)
	self.DELETE("/friends/:userId", friends.Delete)

	self.POST("/requests/:userId/accept", requests.Accept)
	self.POST("/requests/:userId/decline", requests.Decline)
	self.GET("/requests", requests.Get)

	self.PUT("/blocked/:userId", blocked.Create)
	self.GET("/blocked", blocked.Get)
	self.DELETE("/blocked/:userId", blocked.Delete)

	self.GET("/guilds", getSelfGuilds)
	self.DELETE("/guilds/:guildId", leaveGuild)

	self.DELETE("/msgs", clearUserMsg)
}
