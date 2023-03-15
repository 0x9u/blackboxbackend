package users

import (
	"github.com/asianchinaboi/backendserver/internal/api/middleware"
	"github.com/asianchinaboi/backendserver/internal/api/routes/users/blocked"
	"github.com/asianchinaboi/backendserver/internal/api/routes/users/directmsgs"
	"github.com/asianchinaboi/backendserver/internal/api/routes/users/friends"
	"github.com/asianchinaboi/backendserver/internal/api/routes/users/msgs"
	"github.com/asianchinaboi/backendserver/internal/api/routes/users/requests"
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

	users.POST("/@me/dms", directmsgs.Create).Use(middleware.Auth)
	users.GET("/@me/dms", directmsgs.Get).Use(middleware.Auth)
	users.DELETE("/@me/dms/:dmId", directmsgs.Delete).Use(middleware.Auth)

	users.POST("/@me/dms/:dmId/msgs", msgs.Send).Use(middleware.Auth)
	users.GET("/@me/dms/:dmId/msgs", msgs.Get).Use(middleware.Auth)
	users.PATCH("/@me/dms/:dmId/msgs/:msgId", msgs.Edit).Use(middleware.Auth)
	users.DELETE("/@me/dms/:dmId/msgs/:msgId", msgs.Delete).Use(middleware.Auth)
	users.POST("/@me/dms/:dmId/msgs/typing", msgs.Typing).Use(middleware.Auth)

	users.PUT("/@me/friends/:userId", friends.Create).Use(middleware.Auth)
	users.GET("/@me/friends", friends.Get).Use(middleware.Auth)
	users.DELETE("/@me/friends/:userId", friends.Delete).Use(middleware.Auth)

	users.POST("/@me/requests/:userId/accept", requests.Accept).Use(middleware.Auth)
	users.POST("/@me/requests/:userId/decline", requests.Decline).Use(middleware.Auth)
	users.GET("/@me/requests", requests.Get).Use(middleware.Auth)

	users.PUT("/@me/blocked/:userId", blocked.Create).Use(middleware.Auth)
	users.GET("/@me/blocked", blocked.Get).Use(middleware.Auth)
	users.DELETE("/@me/blocked/:userId", blocked.Delete).Use(middleware.Auth)

	users.GET("/@me/guilds", getSelfGuilds).Use(middleware.Auth)
	users.DELETE("/@me/guilds/:guildId", leaveGuild).Use(middleware.Auth)

	users.GET("/:userId", getUserInfo).Use(middleware.Auth)

}
