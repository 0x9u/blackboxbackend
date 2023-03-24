package schedule

import (
	"fmt"
	"os"
	"time"

	"github.com/asianchinaboi/backendserver/internal/config"
	"github.com/asianchinaboi/backendserver/internal/db"
	"github.com/asianchinaboi/backendserver/internal/logger"
)

func deleteTempFile() {
	logger.Info.Println("Deleting temp files")
	fileRows, err := db.Db.Query("DELETE FROM FROM files WHERE temp = 1 AND created_at < $1 RETURNING id", time.Now().Add(-config.Config.Server.TempFileAlive))
	if err != nil {
		logger.Error.Println(err)
		return
	}
	defer fileRows.Close()
	for fileRows.Next() {
		var fileId int64
		if err := fileRows.Scan(&fileId); err != nil {
			logger.Error.Println(err)
			continue
		}
		if err := os.Remove(fmt.Sprintf("uploads/%d.lz4", fileId)); err != nil {
			logger.Warn.Printf("unable to remove file: %v\n", err)
		}
	}
}
