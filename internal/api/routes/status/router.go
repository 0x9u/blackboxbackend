package status

import "github.com/gin-gonic/gin"

func Routes(r *gin.RouterGroup) {
	status := r.Group("/status")
	status.GET("/", ShowStatus)
}
