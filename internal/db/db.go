package db

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"

	"github.com/asianchinaboi/backendserver/internal/config"
	"github.com/asianchinaboi/backendserver/internal/logger"
)

var (
	Db *sql.DB
)

func init() {
	var err error
	loginInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Config.Server.DatabaseConfig.Host,
		config.Config.Server.DatabaseConfig.Port,
		config.Config.Server.DatabaseConfig.User,
		config.Config.Server.DatabaseConfig.Password,
		config.Config.Server.DatabaseConfig.DBName,
		config.Config.Server.DatabaseConfig.SSLMode)
	Db, err = sql.Open("postgres", loginInfo)
	if err != nil {
		logger.Fatal.Println(err)
	}
	Db.SetMaxOpenConns(config.Config.Server.DatabaseConfig.MaxConns)
}
