package guilds

import (
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

/*
alter table userguilds add column owner bool not null default false;
update userguilds  set owner = true from guilds where guild_id = guilds.id and user_id =  guilds.owner_id;
alter table guilds drop column owner_id;
*/

//accepts name, icon, savechat

func createGuild(c *gin.Context) {
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
	var guild events.Guild
	if err := c.ShouldBindJSON(&guild); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusBadRequest, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusBadJSON,
		})
		return
	}

	if statusCode, err := events.ValidateGuildInput(&guild); err != nil {
		if statusCode != errors.StatusInternalError {
			c.JSON(http.StatusUnprocessableEntity, errors.Body{
				Error:  err.Error(),
				Status: statusCode,
			})
			return
		} else {
			c.JSON(http.StatusInternalServerError, errors.Body{
				Error:  err.Error(),
				Status: errors.StatusInternalError,
			})
			return
		}
	}

	var guildId int
	if err := db.Db.QueryRow("INSERT INTO guilds (name, icon, save_chat) VALUES ($1, $2, $3) RETURNING id", guild.Name, guild.Icon, guild.SaveChat).Scan(&guildId); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	invite := events.Invite{
		Invite:  session.GenerateRandString(10),
		GuildId: guildId,
	}

	if _, err := db.Db.Exec("INSERT INTO unreadmessages (guild_id, user_id) VALUES ($1, $2)", guildId, user.Id); err != nil { //cleanup if failed later
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	if _, err := db.Db.Exec("INSERT INTO userguilds (guild_id, user_id, owner) VALUES ($1, $2, true)", guildId, user.Id); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}
	if _, err := db.Db.Exec("INSERT INTO invites (invite, guild_id) VALUES ($1, $2)", invite.Invite, guildId); err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusInternalServerError, errors.Body{
			Error:  err.Error(),
			Status: errors.StatusInternalError,
		})
		return
	}

	res := wsclient.DataFrame{
		Op: wsclient.TYPE_DISPATCH,
		Data: events.Guild{
			GuildId: guildId,
			OwnerId: user.Id,
			Name:    guild.Name,
			Icon:    guild.Icon,
		},
		Event: events.CREATE_GUILD,
	}
	invRes := wsclient.DataFrame{
		Op:    wsclient.TYPE_DISPATCH,
		Data:  invite,
		Event: events.CREATE_INVITE,
	}
	wsclient.Pools.BroadcastClient(user.Id, res)
	//shit i forgot to create a pool
	wsclient.Pools.AddUserToGuildPool(user.Id, guildId)
	wsclient.Pools.BroadcastGuild(guildId, invRes)
	//possible race condition but shouldnt be possible since sql does it by queue
	c.Status(http.StatusNoContent) //writing this code at nearly 12 am gotta keep the grind up
	//dec 9 2022 writing code at nearly 12 am is not good im fixing it rn and holy crap some of the stuff looks shit
}