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
		SELECT udmg.dm_id, udmg.user_id, u.username 
		udm.msg_id AS last_read_msg_id, COUNT(dm.id) filter (WHERE dm.id > udm.msg_id) AS unread_msgs, udm.time
		FROM userdirectmsgsguilds udmg 
		INNER JOIN users u ON u.id = udm.user_id 
		INNER JOIN unreaddirectmsgs udm ON udm.dm_id = udmg.dm_id AND un.user_id = $1
		INNER JOIN directmsgs dm ON dm.id = udm.msg_id 
		WHERE udmg.dm_id IN (SELECT dm_id FROM userdirectmsgsguilds WHERE user_id=$1 AND left_dm = false) AND udmg.user_id != $1
	`, user.Id)
	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	defer rows.Close()
	var openDMs []events.Dm
	for rows.Next() {
		var dm events.Dm
		if err := rows.Scan(&dm.DmId, &dm.UserInfo.UserId, &dm.UserInfo.Name); err != nil {
			logger.Error.Println(err)
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
		openDMs = append(openDMs, dm)
	}
	c.JSON(http.StatusOK, openDMs)
}
