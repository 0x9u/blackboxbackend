package static

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/asianchinaboi/backendserver/internal/errors"
	"github.com/asianchinaboi/backendserver/internal/logger"
	"github.com/gin-gonic/gin"
)

func staticFiles(c *gin.Context) {
	path, err := filepath.Abs(c.Request.URL.Path)
	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError,
			errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
		return
	}
	path = filepath.Join("build", path)
	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		http.ServeFile(c.Writer, c.Request, filepath.Join("build", "index.html"))
		return
	} else if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError,
			errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
		return
	}
	http.FileServer(http.Dir("build")).ServeHTTP(c.Writer, c.Request)
}
