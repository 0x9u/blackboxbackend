package files

import (
	"github.com/gin-gonic/gin"
)

//make this at the end

func Routes(r *gin.RouterGroup) {
	files := r.Group("/files")
	//files.Use(middleware.Auth)
	files.GET("/:entityType/:fileId", get)
}
