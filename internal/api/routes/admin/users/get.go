package users

import (
	"database/sql"
	"net/http"
	"regexp"
	"strconv"

	"github.com/asianchinaboi/backendserver/internal/api/middleware"
	"github.com/asianchinaboi/backendserver/internal/db"
	"github.com/asianchinaboi/backendserver/internal/errors"
	"github.com/asianchinaboi/backendserver/internal/events"
	"github.com/asianchinaboi/backendserver/internal/logger"
	"github.com/asianchinaboi/backendserver/internal/session"
	"github.com/gin-gonic/gin"
)

func Get(c *gin.Context) {
	user := c.MustGet(middleware.User).(*session.Session)
	if user == nil {
		logger.Error.Println("user token not sent in data")
		c.JSON(http.StatusInternalServerError,
			errors.Body{
				Error:  errors.ErrSessionDidntPass.Error(),
				Status: errors.StatusInternalError,
			})
		return
	}
	queryParms := c.Request.URL.Query()
	//GET PAGE NUM
	page := queryParms.Get("page")
	//GET LIMIT
	limit := queryParms.Get("limit")
	if match, err := regexp.MatchString(`^[0-9]+$`, page); !match {
		page = "0"
	} else if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	if match, err := regexp.MatchString(`^[0-9]+$`, limit); !match {
		limit = "0"
	} else if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	intPage, err := strconv.ParseInt(page, 10, 64)
	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	intLimit, err := strconv.ParseInt(limit, 10, 64)
	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	offset := intPage * intLimit
	var nullIntLimit sql.NullInt64
	if limit == "0" {
		nullIntLimit.Valid = false
	} else {
		nullIntLimit.Valid = true
		nullIntLimit.Int64 = intLimit
	}
	logger.Debug.Println(limit, offset)
	//somehow escaping characters probs why
	rows, err := db.Db.Query("SELECT users.id, username, email, files.id FROM users LEFT JOIN files ON files.user_id = users.id LIMIT $1 OFFSET $2", nullIntLimit, offset)
	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	//add page limit
	defer rows.Close()
	var users []events.User
	for rows.Next() {
		var user events.User
		if err := rows.Scan(&user.UserId, &user.Name, &user.Email, &user.ImageId); err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
		users = append(users, user)
	}
	c.JSON(http.StatusOK, users)
}
