package ws

import (
	"github.com/gin-gonic/gin"
)

func Routes(r *gin.RouterGroup) {
	ws := r.Group("/ws")
	ws.GET("/", webSocket)
}
