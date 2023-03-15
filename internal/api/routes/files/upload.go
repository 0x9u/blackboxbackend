package files

import "github.com/gin-gonic/gin"

func upload(c *gin.Context) {
	form, _ := c.MultipartForm()
	files := form.File["files[]"]
	filePaths := []string{}
	for _, file := range files {
		filePath := "uploads/" + file.Filename
		c.SaveUploadedFile(file, filePath)
		filePaths = append(filePaths, filePath)
	}
}
