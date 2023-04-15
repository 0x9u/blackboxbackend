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
	fileRows, err := db.Db.Query("DELETE FROM files WHERE temp = true AND created < $1 RETURNING id, entity_type", time.Now().Add(-config.Config.Server.TempFileAlive))
	if err != nil {
		logger.Error.Println(err)
		return
	}
	defer fileRows.Close()
	for fileRows.Next() {
		var fileId int64
		var entityType string
		if err := fileRows.Scan(&fileId, &entityType); err != nil {
			logger.Error.Println(err)
			continue
		}
		if err := os.Remove(fmt.Sprintf("uploads/%s/%d.lz4", entityType, fileId)); err != nil {
			logger.Warn.Printf("unable to remove file: %v\n", err)
		}
	}
}
