package bans

import (
	"database/sql"
	"net/http"
	"regexp"
	"strconv"

	"github.com/asianchinaboi/backendserver/internal/api/middleware"
	"github.com/asianchinaboi/backendserver/internal/db"
	"github.com/asianchinaboi/backendserver/internal/errors"
	"github.com/asianchinaboi/backendserver/internal/events"
	"github.com/asianchinaboi/backendserver/internal/session"
	"github.com/asianchinaboi/backendserver/internal/wsclient"
	"github.com/gin-gonic/gin"
)

func Ban(c *gin.Context) {
	user := c.MustGet(middleware.User).(*session.Session)
	if user == nil {
		errors.SendErrorResponse(c, errors.ErrSessionDidntPass, errors.StatusInternalError)
		return
	}

	guildId := c.Param("guildId")
	if match, err := regexp.MatchString("^[0-9]+$", guildId); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	} else if !match {
		errors.SendErrorResponse(c, errors.ErrRouteParamInvalid, errors.StatusRouteParamInvalid)
		return
	}

	userId := c.Param("userId")
	if match, err := regexp.MatchString("^[0-9]+$", userId); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	} else if !match {
		errors.SendErrorResponse(c, errors.ErrRouteParamInvalid, errors.StatusRouteParamInvalid)
		return
	}

	intGuildId, err := strconv.ParseInt(guildId, 10, 64)
	if err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}
	intUserId, err := strconv.ParseInt(userId, 10, 64)
	if err != nil {
		errors.SendErrorResponse(c, errors.ErrRouteParamInvalid, errors.StatusRouteParamInvalid)
		return
	}

	if intUserId == user.Id { //failsafe or something
		errors.SendErrorResponse(c, errors.ErrCantKickBanSelf, errors.StatusCantKickBanSelf)
		return
	}

	var hasAuth bool
	var isUserAuth bool
	var isBanned bool
	var userExists bool
	var isDm bool

	if err := db.Db.QueryRow(`
	SELECT EXISTS (SELECT 1 FROM userguilds WHERE guild_id=$1 AND user_id=$2 AND (admin=true OR owner=true)), 
	EXISTS (SELECT 1 FROM userguilds WHERE guild_id = $1 AND user_id=$3 AND banned = true), 
	EXISTS (SELECT 1 FROM guilds WHERE id = $1 AND dm = true),
	EXISTS (SELECT 1 FROM userguilds WHERE guild_id = $1 AND user_id = $3 AND (admin = true OR owner=true)),
	EXISTS (SELECT 1 FROM users WHERE id = $3)
	`, guildId, user.Id, userId).Scan(&hasAuth, &isBanned, &isDm, &isUserAuth, &userExists); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	if isDm {
		errors.SendErrorResponse(c, errors.ErrGuildIsDm, errors.StatusGuildIsDm)
		return
	}

	if isBanned {
		errors.SendErrorResponse(c, errors.ErrAlreadyBanned, errors.StatusAlreadyBanned)
		return
	}
	if !hasAuth || isUserAuth {
		errors.SendErrorResponse(c, errors.ErrNotGuildAuthorised, errors.StatusNotGuildAuthorised)
		return
	}

	if !userExists {
		errors.SendErrorResponse(c, errors.ErrUserNotFound, errors.StatusUserNotFound)
		return
	}

	if _, err := db.Db.Exec("INSERT INTO userguilds (guild_id, user_id, banned) VALUES ($1, $2, true) ON CONFLICT (guild_id, user_id) DO UPDATE SET banned=true", guildId, userId); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	var username string
	var imageId sql.NullInt64
	if err := db.Db.QueryRow("SELECT username, files.id FROM users LEFT JOIN files ON files.user_id = users.id WHERE users.id=$1", userId).Scan(&username, &imageId); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	var resImageId int64
	if imageId.Valid {
		resImageId = imageId.Int64
	} else {
		resImageId = -1
	}

	rows, err := db.Db.Query("SELECT user_id FROM userguilds WHERE guild_id = $1 AND (owner = true OR admin = true)", guildId)
	if err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var adminUserId int64
		if err := rows.Scan(&adminUserId); err != nil {
			errors.SendErrorResponse(c, err, errors.StatusInternalError)
			return
		}
		res := wsclient.DataFrame{
			Op: wsclient.TYPE_DISPATCH,
			Data: events.Member{
				GuildId: intGuildId,
				UserInfo: events.User{
					UserId:  intUserId,
					Name:    username,
					ImageId: resImageId,
				},
			},
			Event: events.ADD_USER_BANLIST,
		}
		wsclient.Pools.BroadcastClient(adminUserId, res)
	}

	banRes := wsclient.DataFrame{
		Op: wsclient.TYPE_DISPATCH,
		Data: events.Guild{
			GuildId: intGuildId,
		},
		Event: events.DELETE_GUILD,
	}
	guildRes := wsclient.DataFrame{
		Op: wsclient.TYPE_DISPATCH,
		Data: events.Member{
			GuildId: intGuildId,
			UserInfo: events.User{
				UserId: intUserId,
			},
		},
		Event: events.REMOVE_USER_GUILDLIST,
	}
	wsclient.Pools.BroadcastClient(intUserId, banRes)
	wsclient.Pools.RemoveUserFromGuildPool(intGuildId, intUserId)
	wsclient.Pools.BroadcastGuild(intGuildId, guildRes)
	c.Status(http.StatusNoContent)
}
