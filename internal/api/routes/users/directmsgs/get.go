package directmsgs

import (
	"net/http"

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

	rows, err := db.Db.Query(`
		SELECT ud.sender_id AS friend_id, u.username AS username FROM userdirectmsgs ud INNER JOIN users u ON ud.sender_id = u.id WHERE ud.receiver_id = $1
		 UNION 
		SELECT ud.receiver_id AS friend_id, u.username AS username FROM userdirectmsgs ud INNER JOIN users u ON ud.receiver_id = u.id WHERE ud.sender_id = $1
	`, user.Id)
	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	var openDMs []events.User
	for rows.Next() {
		var friendUser events.User
		if err := rows.Scan(&friendUser.UserId, &friendUser.Name); err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
		openDMs = append(openDMs, friendUser)
	}
	c.JSON(http.StatusOK, openDMs)
}
