package guilds

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/asianchinaboi/backendserver/internal/api/middleware"
	"github.com/asianchinaboi/backendserver/internal/db"
	"github.com/asianchinaboi/backendserver/internal/errors"
	"github.com/asianchinaboi/backendserver/internal/events"
	"github.com/asianchinaboi/backendserver/internal/logger"
	"github.com/asianchinaboi/backendserver/internal/session"
	"github.com/asianchinaboi/backendserver/internal/wsclient"
	"github.com/gin-gonic/gin"
)

type joinGuildBody struct {
	Invite string `json:"invite" binding:"required"`
}

func joinGuild(c *gin.Context) {
	user := c.MustGet(middleware.User).(*session.Session)
	if user == nil {
		errors.SendErrorResponse(c, errors.ErrSessionDidntPass, errors.StatusInternalError)
		return
	}
	var invite joinGuildBody

	if err := c.ShouldBindJSON(&invite); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}
	if invite.Invite == "" {
		errors.SendErrorResponse(c, errors.ErrNoInvite, errors.StatusNoInvite)
		return
	}

	var guild events.Guild
	var imageId sql.NullInt64

	err := db.Db.QueryRow(
		`
		SELECT i.guild_id, g.name, f.id, ug.user_id
		FROM invites i INNER JOIN guilds g ON g.id = i.guild_id 
		INNER JOIN userguilds ug ON ug.guild_id = g.id AND owner = true 
		LEFT JOIN files f ON f.guild_id = g.id 
		WHERE i.invite = $1`,
		invite.Invite).Scan(&guild.GuildId, &guild.Name, &imageId, &guild.OwnerId)
	if err != nil && err == sql.ErrNoRows {
		errors.SendErrorResponse(c, errors.ErrInvalidInvite, errors.StatusInvalidInvite)
		return
	} else if err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	if imageId.Valid {
		guild.ImageId = imageId.Int64
	} else {
		guild.ImageId = -1
	}

	var isInGuild bool

	if err := db.Db.QueryRow("SELECT EXISTS (SELECT * FROM userguilds WHERE user_id=$1 AND guild_id=$2)", user.Id, guild.GuildId).Scan(&isInGuild); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}
	if isInGuild {
		errors.SendErrorResponse(c, errors.ErrAlreadyInGuild, errors.StatusAlreadyInGuild)
		return
	}

	logger.Debug.Println("user", user.Id, "joined guild", guild.GuildId)

	//BEGIN TRANSACTION
	ctx := context.Background()
	tx, err := db.Db.BeginTx(ctx, nil)
	if err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}
	defer tx.Rollback() //rollback changes if failed

	if _, err := tx.ExecContext(ctx, "INSERT INTO userguilds (guild_id, user_id) VALUES ($1, $2)", guild.GuildId, user.Id); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	if _, err := tx.ExecContext(ctx, "INSERT INTO unreadmsgs (guild_id, user_id) VALUES ($1, $2)", guild.GuildId, user.Id); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	if err := tx.Commit(); err != nil { //commits the transaction
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	res := wsclient.DataFrame{
		Op:    wsclient.TYPE_DISPATCH,
		Data:  guild,
		Event: events.CREATE_GUILD,
	}
	wsclient.Pools.BroadcastClient(user.Id, res)

	userData := events.Member{} //change name later

	//reusing same imageid from before

	if err := db.Db.QueryRow("SELECT username, files.id FROM users LEFT JOIN files ON files.user_id = users.id WHERE users.id=$1", user.Id).Scan(&userData.UserInfo.Name, &imageId); err != nil {
		errors.SendErrorResponse(c, err, errors.StatusInternalError)
		return
	}

	if imageId.Valid {
		userData.UserInfo.ImageId = imageId.Int64
	} else {
		userData.UserInfo.ImageId = -1
	}

	userData.UserInfo.UserId = user.Id
	userData.GuildId = guild.GuildId

	guildRes := wsclient.DataFrame{
		Op:    wsclient.TYPE_DISPATCH,
		Data:  userData,
		Event: events.ADD_USER_GUILDLIST,
	}
	wsclient.Pools.BroadcastGuild(guild.GuildId, guildRes)
	wsclient.Pools.AddUserToGuildPool(guild.GuildId, user.Id)
	c.Status(http.StatusNoContent)
}
