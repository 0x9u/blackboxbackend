package guilds

import (
	"github.com/asianchinaboi/backendserver/internal/api/middleware"
	"github.com/asianchinaboi/backendserver/internal/api/routes/guilds/admins"
	"github.com/asianchinaboi/backendserver/internal/api/routes/guilds/bans"
	"github.com/asianchinaboi/backendserver/internal/api/routes/guilds/invites"
	"github.com/asianchinaboi/backendserver/internal/api/routes/guilds/members"
	"github.com/asianchinaboi/backendserver/internal/api/routes/guilds/msgs"
	"github.com/gin-gonic/gin"
)

func Routes(r *gin.RouterGroup) {
	guilds := r.Group("/guilds")
	guilds.Use(middleware.Auth)

	guilds.POST("/", createGuild)
	guilds.POST("/join", joinGuild)

	guilds.DELETE("/:guildId", deleteGuild)
	guilds.PATCH("/:guildId", editGuild)
	guilds.GET("/:guildId", getGuild)

	guilds.GET("/:guildId/members", members.Get)
	guilds.DELETE("/:guildId/members/:userId", members.Kick)

	guilds.PUT("/:guildId/admins/:userId", admins.Create)
	guilds.DELETE("/:guildId/admins/:userId", admins.Delete)

	guilds.GET("/:guildId/msgs", msgs.Get)
	guilds.POST("/:guildId/msgs", msgs.Send)
	guilds.DELETE("/:guildId/msgs/:msgId", msgs.Delete)
	guilds.PATCH("/:guildId/msgs/:msgId", msgs.Edit)
	guilds.DELETE("/:guildId/msgs/clear", msgs.Clear)
	guilds.POST("/:guildId/msgs/typing", msgs.Typing) //need to persist to typing in guild pool later on
	guilds.POST("/:guildId/msgs/read", msgs.Read)

	guilds.GET("/:guildId/bans", bans.Get)
	guilds.PUT("/:guildId/bans/:userId", bans.Ban)
	guilds.DELETE("/:guildId/bans/:userId", bans.Unban)

	guilds.GET("/:guildId/invites", invites.Get)
	guilds.POST("/:guildId/invites", invites.Create)
	guilds.DELETE(`/:guildId/invites/:invite`, invites.Delete)
}
