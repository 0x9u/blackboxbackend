package files

import (
	"github.com/gin-gonic/gin"
)

//make this at the end

func Routes(r *gin.RouterGroup) {
	files := r.Group("/files")
	files.GET("/:fileId", get)
}
