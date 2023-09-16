package blocked

import (
	"database/sql"
	"net/http"

	"github.com/asianchinaboi/backendserver/internal/api/middleware"
	"github.com/asianchinaboi/backendserver/internal/db"
	"github.com/asianchinaboi/backendserver/internal/errors"
	"github.com/asianchinaboi/backendserver/internal/events"
	"github.com/asianchinaboi/backendserver/internal/session"
	"github.com/gin-gonic/gin"
)

func Get(c *gin.Context) {
	user := c.MustGet(middleware.User).(*session.Session)
	if user == nil {
		errors.SendErrorResponse(c, errors.ErrSessionDidntPass, errors.StatusInternalError)
		return
	}

	rows, err := db.Db.Query("SELECT b.blocked_id, u.username, f.id FROM blocked b INNER JOIN users u ON b.blocked_id = u.id LEFT JOIN files f ON f.user_id = u.id WHERE b.user_id = $1", user.Id)
	if err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}
	blockedUsers := []events.User{}
	defer rows.Close()
	for rows.Next() {
		blockedUser := events.User{}
		var imageId sql.NullInt64
		if err := rows.Scan(&blockedUser.UserId, &blockedUser.Name, &imageId); err != nil {
			errors.SendErrorResponse(c, err, errors.StatusInternalError)
			return
		}
		if imageId.Valid {
			blockedUser.ImageId = imageId.Int64
		} else {
			blockedUser.ImageId = -1
		}
		blockedUsers = append(blockedUsers, blockedUser)
	}
	c.JSON(http.StatusOK, blockedUsers)
}
