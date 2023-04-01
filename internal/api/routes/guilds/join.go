package guilds

import (
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
	Invite string `json:"invite"`
}

func joinGuild(c *gin.Context) {
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
	var invite joinGuildBody

	if err := c.ShouldBindJSON(&invite); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusBadRequest, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	if invite.Invite == "" {
		logger.Error.Println(errors.ErrNoInvite)
		c.Status(http.StatusUnprocessableEntity)
		return
	}
	row := db.Db.QueryRow(
		`
		SELECT i.guild_id, g.name, g.icon, ug.user_id
		FROM invites i INNER JOIN guilds g ON g.id = i.guild_id 
		INNER JOIN userguilds ug ON ug.guild_id = g.id AND owner = true
		WHERE i.invite = $1`,
		invite.Invite)
	if err := row.Err(); err == sql.ErrNoRows {
		logger.Error.Println(errors.ErrInvalidInvite)
		c.JSON(http.StatusForbidden, errors.Body{
			Error:  errors.ErrInvalidInvite.Error(),
			Status: errors.StatusInvalidInvite,
		})
		return
	} else if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	var guild events.Guild
	row.Scan(&guild.GuildId, &guild.Name, &guild.Icon, &guild.OwnerId)

	var isInGuild bool

	if err := db.Db.QueryRow("SELECT EXISTS (SELECT * FROM userguilds WHERE user_id=$1 AND guild_id=$2)", user.Id, guild.GuildId).Scan(&isInGuild); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	if isInGuild {
		logger.Error.Println(errors.ErrAlreadyInGuild) //applies for banned too
		c.JSON(http.StatusBadRequest, errors.Body{
			Error:  errors.ErrAlreadyInGuild.Error(),
			Status: errors.StatusAlreadyInGuild,
		})
		return
	}

	if _, err := db.Db.Exec("INSERT INTO userguilds (guild_id, user_id) VALUES ($1, $2)", guild.GuildId, user.Id); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	res := wsclient.DataFrame{
		Op:    wsclient.TYPE_DISPATCH,
		Data:  guild,
		Event: events.CREATE_GUILD,
	}
	wsclient.Pools.BroadcastClient(user.Id, res)

	var userData events.UserGuild //change name later
	if err := db.Db.QueryRow("SELECT username FROM users WHERE id=$1", user.Id).Scan(&userData.UserData.Name); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	userData.UserId = user.Id
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
