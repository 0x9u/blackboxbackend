package admin

import (
	"fmt"
	"net/http"

	"github.com/asianchinaboi/backendserver/internal/api/middleware"
	"github.com/asianchinaboi/backendserver/internal/db"
	"github.com/asianchinaboi/backendserver/internal/errors"
	"github.com/asianchinaboi/backendserver/internal/session"
	"github.com/gin-gonic/gin"
)

type sqlQueryBody struct {
	Query string `json:"query"`
}

type sqlQueryRes struct {
	Columns []string   `json:"columns"`
	Rows    [][]string `json:"rows"`
}

func runSqlQuery(c *gin.Context) {
	user := c.MustGet(middleware.User).(*session.Session)
	if user == nil {
		errors.SendErrorResponse(c, errors.ErrSessionDidntPass, errors.StatusInternalError)
		return
	}
	if !user.Perms.Admin {
		errors.SendErrorResponse(c, errors.ErrNotAuthorised, errors.StatusNotAuthorised)
		return
	}
	var sqlQuery sqlQueryBody
	if err := c.ShouldBindJSON(&sqlQuery); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusBadRequest)
		return
	}

	rows, err := db.Db.Query(sqlQuery.Query)
	if err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	res := sqlQueryRes{}
	res.Columns = columns

	if len(columns) > 0 {
		values := make([]interface{}, len(columns))
		pointers := make([]interface{}, len(columns))
		for i := range values {
			pointers[i] = &values[i]
		}

		for rows.Next() {
			err := rows.Scan(pointers...)
			if err != nil {
				errors.SendErrorResponse(c, err, errors.StatusInternalError)
				return
			}
			row := []string{}
			for _, value := range values {
				strValue := fmt.Sprintf("%v", value)
				row = append(row, strValue)
			}
			res.Rows = append(res.Rows, row)
		}
	}
	c.JSON(http.StatusOK, res)
}
