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

	//i forgot how this works - supposed to show unread messages + mention count and sort descending from last person user chatted with
	rows, err := db.Db.Query(`
		SELECT udmg.dm_id, udmg.receiver_id, u.username,
		udm.msg_id AS last_read_msg_id, COUNT(dm.id) filter (WHERE dm.id > udm.msg_id) AS unread_msgs,
		COUNT(dmm.msg_id) filter (WHERE dmm.user_id = $1 AND dmm.msg_id > udm.msg_id) + 
		COUNT(dm.msg_id) filter (WHERE dm.mentions_everyone = true AND dm.id > udm.msg_id) AS mentions, udm.time
		FROM userdirectmsgsguilds udmg 
		INNER JOIN users u ON u.id = udmg.receiver_id
		INNER JOIN unreaddirectmsgs udm ON udm.dm_id = udmg.dm_id AND udm.user_id = $1
		LEFT JOIN directmsgs dm ON dm.id = udm.msg_id 
		LEFT JOIN directmsgmentions dmm ON dmm.msg_id = dm.id 
		WHERE udmg.user_id=$1 AND udmg.left_dm = false
		GROUP BY udmg.dm_id, udmg.receiver_id, u.username, udm.time
		ORDER BY udm.time DESC
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
		if err := rows.Scan(&dm.DmId, &dm.UserInfo.UserId, &dm.UserInfo.Name, &dm.UnreadMsg.Id, &dm.UnreadMsg.Count, &dm.UnreadMsg.Mentions, &dm.UnreadMsg.Time ); err != nil {
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
