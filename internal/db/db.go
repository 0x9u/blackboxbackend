package db

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"

	"github.com/asianchinaboi/backendserver/internal/logger"
)

var (
	Db *sql.DB
)

func init() {
	var err error
	loginInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s", host, port, user, password, dbname, sslenabled)
	Db, err = sql.Open("postgres", loginInfo)
	if err != nil {
		logger.Fatal.Println(err)
	}
}
