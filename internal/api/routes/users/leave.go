package users

import (
	"context"
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

func leaveGuild(c *gin.Context) {
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

	var isOwner bool
	var isInGuild bool

	if err := db.Db.QueryRow(`
	SELECT EXISTS (SELECT 1 FROM userguilds WHERE guild_id=$1 AND user_id=$2 AND owner = true),
	EXISTS(SELECT 1 FROM userguilds WHERE guild_id=$1 AND user_id=$2 AND banned = false)
	`, guildId, user.Id).Scan(&isOwner, &isInGuild); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	if isOwner {
		errors.SendErrorResponse(c, errors.ErrCantLeaveOwnGuild, errors.StatusCantLeaveOwnGuild)
		return
	}
	if !isInGuild {
		errors.SendErrorResponse(c, errors.ErrNotInGuild, errors.StatusNotInGuild)
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

	if _, err := tx.ExecContext(ctx, "DELETE FROM userguilds WHERE guild_id=$1 AND user_id=$2", guildId, user.Id); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM unreadmsgs WHERE guild_id=$1 AND user_id=$2", guildId, user.Id); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	if err := tx.Commit(); err != nil { //commits the transaction
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	intGuildId, err := strconv.ParseInt(guildId, 10, 64)
	if err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	res := wsclient.DataFrame{
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
				UserId: user.Id,
			},
		},
		Event: events.REMOVE_USER_GUILDLIST,
	}
	wsclient.Pools.BroadcastClient(user.Id, res)
	wsclient.Pools.RemoveUserFromGuildPool(intGuildId, user.Id)
	wsclient.Pools.BroadcastGuild(intGuildId, guildRes)
	c.Status(http.StatusNoContent)
}
