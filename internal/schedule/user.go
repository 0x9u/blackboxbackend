package schedule

import (
	"time"

	"github.com/asianchinaboi/backendserver/internal/db"
	"github.com/asianchinaboi/backendserver/internal/logger"
)

func deleteTokens() {
	if _, err := db.Db.Exec("DELETE FROM tokens WHERE token_expires < $1", time.Now().Unix()); err != nil {
		logger.Error.Println(err)
	}
}
