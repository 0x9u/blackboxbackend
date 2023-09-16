package static

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/asianchinaboi/backendserver/internal/errors"
	"github.com/gin-gonic/gin"
)

func staticFiles(c *gin.Context) {
	path, err := filepath.Abs(c.Request.URL.Path)
	if err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}
	path = filepath.Join("build", path)
	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		http.ServeFile(c.Writer, c.Request, filepath.Join("build", "index.html"))
		return
	} else if err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}
	http.FileServer(http.Dir("build")).ServeHTTP(c.Writer, c.Request)
}
