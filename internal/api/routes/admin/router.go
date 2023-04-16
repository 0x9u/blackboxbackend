package admin

import (
	"github.com/asianchinaboi/backendserver/internal/api/middleware"
	"github.com/asianchinaboi/backendserver/internal/api/routes/admin/guilds"
	"github.com/asianchinaboi/backendserver/internal/api/routes/admin/guilds/bans"
	"github.com/asianchinaboi/backendserver/internal/api/routes/admin/guilds/members"
	"github.com/asianchinaboi/backendserver/internal/api/routes/admin/users"
	"github.com/gin-gonic/gin"
)

//make this at the end

func Routes(r *gin.RouterGroup) {
	admin := r.Group("/admin")
	admin.Use(middleware.Auth)
	//ADMIN ONLY
	admin.POST("/reset", reset)     //extremely dangerous
	admin.POST("/sql", runSqlQuery) //extremely dangeorus too
	//ADMIN ONLY

	admin.POST("/banip", banIP)

	admin.GET("/users", users.Get) //two query params page and limit
	admin.DELETE("/users/:userId", users.Delete)
	admin.PATCH("/users/:userId", users.Edit)

	admin.GET("/guilds", guilds.Get) //two query params page and limit
	admin.DELETE("/guilds/:guildId", guilds.Delete)
	admin.PATCH("/guilds/:guildId", guilds.Edit)

	admin.GET("/guilds/:guildId/members", members.Get)
	admin.DELETE("/guilds/:guildId/members/:userId", members.Kick)

	admin.GET("/guilds/:guildId/bans", bans.Get)
	admin.PUT("/guilds/:guildId/bans/:userId", bans.Ban)
	admin.DELETE("/guilds/:guildId/bans/:userId", bans.Unban)
}
