package users

import (
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
	intPage, err := strconv.Atoi(page)
	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	intLimit, err := strconv.Atoi(limit)
	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	offset := intPage * intLimit
	if limit == "0" {
		limit = "ALL"
	}
	logger.Debug.Println(limit, offset)
	//somehow escaping characters probs why
	rows, err := db.Db.Query("SELECT id, username, email FROM users LIMIT $1 OFFSET $2", limit, offset)
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
		if err := rows.Scan(&user.UserId, &user.Name, &user.Email); err != nil {
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
