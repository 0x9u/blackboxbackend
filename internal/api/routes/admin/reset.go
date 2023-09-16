package admin

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/asianchinaboi/backendserver/internal/api/middleware"
	"github.com/asianchinaboi/backendserver/internal/db"
	"github.com/asianchinaboi/backendserver/internal/errors"
	"github.com/asianchinaboi/backendserver/internal/logger"
	"github.com/asianchinaboi/backendserver/internal/session"
	"github.com/asianchinaboi/backendserver/internal/wsclient"
	"github.com/gin-gonic/gin"
)

//extremely dangerous
//with great power comes with great responsibility
//you have been warned

type fileEntity struct {
	Id         int64
	EntityType string
}

func reset(c *gin.Context) {
	user := c.MustGet(middleware.User).(*session.Session)
	if user == nil {
		errors.SendErrorResponse(c, errors.ErrSessionDidntPass, errors.StatusInternalError)
		return
	}
	if !user.Perms.Admin {
		errors.SendErrorResponse(c, errors.ErrNotAuthorised, errors.StatusNotAuthorised)
		return
	}

	//BEGIN TRANSACTION
	ctx := context.Background()
	tx, err := db.Db.BeginTx(ctx, nil)
	if err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}
	defer tx.Rollback() //rollback changes if failed

	var files []fileEntity
	fileRows, err := db.Db.Query("SELECT id, entity_type FROM files")
	if err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}
	for fileRows.Next() {
		file := fileEntity{}
		if err := fileRows.Scan(&file.Id, &file.EntityType); err != nil {
			errors.SendErrorResponse(c, err, errors.StatusInternalError)
			return
		}
		files = append(files, file)
	}
	fileRows.Close()
	if _, err := tx.ExecContext(ctx, "DELETE FROM guilds"); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM users"); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM bannedips"); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}
	if err := tx.Commit(); err != nil { //commits the transaction
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	wsclient.Pools.RemoveAll()

	for _, file := range files {
		if err := os.Remove(fmt.Sprintf("uploads/%s/%d.lz4", file.EntityType, file.Id)); err != nil {
			logger.Warn.Printf("unable to remove file: %v\n", err)
		}
	}

	logger.Warn.Println("And thus the database is now gone I hope you're happy")
	logger.Warn.Println("If you didn't do this u fucked up big time lmao") //cool and extremely helpful message
	c.Status(http.StatusNoContent)
}
