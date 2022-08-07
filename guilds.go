package main

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/asianchinaboi/backendserver/logger"
)

/*
Guilds table
id PRIMARY KEY SERIAL | name VARCHAR(16) | icon INT | owner_id INT | invites_amount SMALLINT DEFAULT 1 | public BOOLEAN DEFAULT FALSE | keephistory BOOLEAN DEFAULT FALSE
*/
/*
Invites table
invite VARCHAR(10) | guild_id INT | ExpireDate (IMPLEMENT THIS LATER ON) INT64
*/
type createGuildUploadData struct { //use annoymous structs next time
	Icon     int    `json:"Icon"` // if icon none its zero assume no icon
	Name     string `json:"Name"`
	SaveChat bool   `json:"SaveChat"`
}

type getGuildSettingData struct { //implement settings a normal user is unable to see
	SaveChat bool `json:"SaveChat"`
}

type joinGuildUploadData struct { //same to you join guild
	Invite string `json:"Invite"`
}

//small but keep it like that for now
type changeGuild struct {
	Guild    int    `json:"Guild"`
	SaveChat bool   `json:"SaveChat"`
	Name     string `json:"Name"`
	Icon     int    `json:"Icon"`
}

type changeGuildUser struct { //for the users not the owner
	Name  string
	Guild int
	Icon  int
}

type userGuild struct {
	Username string `json:"Username"`
	Id       int    `json:"Id"`
	Icon     int    `json:"Icon"`
}

type userGuildAdd struct {
	Guild int       `json:"Guild"`
	User  userGuild `json:"User"`
}

type userBannedAdd userGuildAdd

type userGuildRemove struct {
	Guild int `json:"Guild"`
	Id    int `json:"Id"` //Id to remove user
}

type userBannedRemove userGuildRemove

type infoGuild struct {
	Owner int    `json:"Owner"`
	Id    int    `json:"Id"`
	Name  string `json:"Name"`
	Icon  int    `json:"Icon"`
}

type joinGuildData infoGuild

type leaveGuildData struct {
	Guild int `json:"Guild"`
}

type deleteGuildData leaveGuildData

type kickBanUserData struct {
	Id    int `json:"Id"`
	Guild int `json:"Guild"`
}

type unbanUserData kickBanUserData

func createGuild(w http.ResponseWriter, r *http.Request, user *session) {

	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}
	var guild createGuildUploadData
	if err := json.Unmarshal(bodyBytes, &guild); err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}
	var guild_id int
	row := db.QueryRow("INSERT INTO guilds (name, icon, owner_id, save_chat) VALUES ($1, $2, $3, $4) RETURNING id", guild.Name, guild.Icon, user.Id, guild.SaveChat)
	row.Scan(&guild_id)
	invite := inviteAdded{
		Invite: generateRandString(10),
	}
	if _, err = db.Exec("INSERT INTO userguilds (guild_id, user_id) VALUES ($1, $2)", guild_id, user.Id); err != nil { //cleanup if failed later
		reportError(http.StatusBadRequest, w, err)
		return
	}
	if _, err = db.Exec("INSERT INTO invites (invite, guild_id) VALUES ($1, $2)", invite.Invite, guild_id); err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}
	if err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}
	broadcastClient(user.Id, joinGuildData{
		Id:    guild_id,
		Owner: user.Id,
		Name:  guild.Name,
		Icon:  guild.Icon,
	})
	//shit i forgot to create a pool
	createPool(guild_id)
	addUserToPool(guild_id, user.Id)
	broadcastGuild(guild_id, invite)
	//possible race condition but shouldnt be possible since sql does it by queue
	w.WriteHeader(http.StatusOK) //writing this code at nearly 12 am gotta keep the grind up
	//WORKS BABAY FUCK YEAH WOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOoo
}

func deleteGuild(w http.ResponseWriter, r *http.Request, user *session) {
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}
	var guild deleteGuildData
	err = json.Unmarshal(bodyBytes, &guild)
	if err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}
	var valid bool
	row := db.QueryRow("SELECT EXISTS (SELECT id FROM guilds WHERE id=$1 AND owner_id=$2)", guild.Guild, user.Id)
	if err := row.Scan(&valid); err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	if !valid { //user is a fraud
		reportError(http.StatusBadRequest, w, errorNotGuildOwner)
		return
	}
	_, err = db.Exec("DELETE FROM messages WHERE guild_id=$1", guild.Guild) //start deleting
	if err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	_, err = db.Exec("DELETE FROM invites WHERE guild_id=$1", guild.Guild) //delete all existing invites
	if err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	_, err = db.Exec("DELETE FROM userguilds WHERE guild_id=$1", guild.Guild) //delete from users guilds
	if err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	_, err = db.Exec("DELETE FROM guilds WHERE id=$1", guild.Guild)
	if err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	broadcastGuild(guild.Guild, leaveGuildData(guild)) // kick everyone out of the guild
	w.WriteHeader(http.StatusOK)
}

func getGuild(w http.ResponseWriter, r *http.Request, user *session) {
	log.WriteLog(logger.INFO, "Getting guilds")
	rows, err := db.Query(
		`
		SELECT g.id, g.name, g.icon, g.owner_id
		FROM userguilds u 
		INNER JOIN guilds g ON g.id = u.guild_id WHERE u.user_id=$1 AND u.banned = false`,
		user.Id,
	)
	if err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	guilds := []infoGuild{}
	for rows.Next() {
		var guild infoGuild
		err = rows.Scan(&guild.Id, &guild.Name, &guild.Icon, &guild.Owner)
		if err != nil {
			reportError(http.StatusInternalServerError, w, err)
			return
		}
		guilds = append(guilds, guild)
	}
	bodyBytes, err := json.Marshal(guilds)
	if err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(bodyBytes)
}

func getGuildUsers(w http.ResponseWriter, r *http.Request, user *session) {
	params := r.URL.Query()
	guild, err := strconv.Atoi(params.Get("guild"))
	if err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}

	var valid bool
	row := db.QueryRow("SELECT EXISTS (SELECT * FROM userguilds WHERE guild_id=$1 AND user_id=$2 AND banned=false)", guild, user.Id)
	if err := row.Scan(&valid); err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	if !valid {
		reportError(http.StatusInternalServerError, w, errorNotInGuild)
		return
	}
	rows, err := db.Query("SELECT users.username, users.id FROM userguilds INNER JOIN users ON userguilds.user_id=users.id WHERE guild_id=$1 AND banned = false", guild)
	if err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	userlist := []userGuild{}
	for rows.Next() {
		var user userGuild
		err = rows.Scan(&user.Username, &user.Id)
		user.Icon = 0 //placeholder until upload is implemented
		if err != nil {
			reportError(http.StatusInternalServerError, w, err)
			return
		}
		userlist = append(userlist, user)
	}
	bodyBytes, err := json.Marshal(userlist)
	if err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(bodyBytes)
}

func editGuild(w http.ResponseWriter, r *http.Request, user *session) {
	var newSettings changeGuild
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}
	err = json.Unmarshal(bodyBytes, &newSettings) //a error really shouldnt occur here i am only
	//going to remove this after i know this works
	if err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	_, err = db.Exec(
		"UPDATE guilds SET save_chat=$1, name=$2, icon=$3 WHERE id=$4 AND owner_id=$5",
		newSettings.SaveChat,
		newSettings.Name,
		newSettings.Icon,
		newSettings.Guild,
		user.Id,
	)
	if err != nil && err != sql.ErrNoRows {
		reportError(http.StatusBadRequest, w, err)
		return
	} else if err == sql.ErrNoRows {
		reportError(http.StatusBadRequest, w, errorNotGuildOwner)
		return
	}

	broadcastGuild(newSettings.Guild, changeGuildUser{Guild: newSettings.Guild, Icon: newSettings.Icon, Name: newSettings.Name})
	w.WriteHeader(http.StatusOK)
}

func joinGuild(w http.ResponseWriter, r *http.Request, user *session) {
	//params := r.URL.Query()
	//invite := params.Get("invite")
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}
	var invite joinGuildUploadData
	err = json.Unmarshal(bodyBytes, &invite)
	if err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}
	if invite.Invite == "" {
		reportError(http.StatusBadRequest, w, errorNoInvite)
		return
	}
	row := db.QueryRow(
		`
		SELECT i.guild_id, g.name, g.icon, g.owner_id 
		FROM invites i INNER JOIN guilds g ON g.id = i.guild_id 
		WHERE i.invite = $1`,
		invite.Invite)
	if err := row.Err(); err == sql.ErrNoRows {
		reportError(http.StatusBadRequest, w, errorInvalidInvite)
		return
	} else if err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	var guild joinGuildData
	row.Scan(&guild.Id, &guild.Name, &guild.Icon, &guild.Owner)

	row = db.QueryRow(
		`
		SELECT EXISTS (SELECT * FROM userguilds WHERE user_id=$1 AND guild_id=$2)
		`,
		user.Id,
		guild.Id,
	)

	var check bool

	if err := row.Scan(&check); err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	if check {
		reportError(http.StatusBadRequest, w, errorAlreadyInGuild)
		return
	}
	_, err = db.Exec("INSERT INTO userguilds (guild_id, user_id) VALUES ($1, $2) ", guild.Id, user.Id)
	if err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	broadcastClient(user.Id, guild)

	var username string
	err = db.QueryRow("SELECT username FROM users WHERE id=$1", user.Id).Scan(&username)
	if err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	//create pool if doesnt exist

	lockPool.Lock()
	_, ok := pools[guild.Id]
	if !ok {
		createPool(guild.Id)
	}
	lockPool.Unlock()
	addUserToPool(guild.Id, user.Id)
	broadcastGuild(guild.Id, userGuildAdd{Guild: guild.Id, User: userGuild{Username: username, Id: user.Id}})
	w.WriteHeader(http.StatusOK)
}

func kickGuildUser(w http.ResponseWriter, r *http.Request, user *session) {
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}
	var kick kickBanUserData
	err = json.Unmarshal(bodyBytes, &kick)
	if err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}
	if kick.Id == 0 {
		reportError(http.StatusBadRequest, w, errorNotInGuild)
		return
	}
	if kick.Id == user.Id {
		reportError(http.StatusBadRequest, w, errorCantKickBanSelf)
		return
	}
	if kick.Guild == 0 {
		reportError(http.StatusBadRequest, w, errorGuildNotProvided)
		return
	}
	var valid bool
	row := db.QueryRow("SELECT EXISTS (SELECT * FROM guilds WHERE id=$1 AND owner_id=$2)", kick.Guild, user.Id)
	if err := row.Scan(&valid); err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	if !valid {
		reportError(http.StatusBadRequest, w, errorNotGuildOwner)
		return
	}
	_, err = db.Exec("DELETE FROM userguilds WHERE guild_id=$1 AND user_id=$2", kick.Guild, kick.Id)
	if err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	broadcastClient(kick.Id, leaveGuildData{Guild: kick.Guild})
	removeUserFromPool(kick.Guild, kick.Id)
	broadcastGuild(kick.Guild, userGuildRemove{Guild: kick.Guild, Id: kick.Id})
	w.WriteHeader(http.StatusOK)
}

func banGuildUser(w http.ResponseWriter, r *http.Request, user *session) {
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}
	var ban kickBanUserData
	err = json.Unmarshal(bodyBytes, &ban)
	if err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}

	if ban.Guild == 0 {
		reportError(http.StatusBadRequest, w, errorGuildNotProvided)
		return
	}

	if ban.Id == 0 {
		reportError(http.StatusBadRequest, w, errorNotInGuild)
		return
	}

	if ban.Id == user.Id { //failsafe or something
		reportError(http.StatusBadRequest, w, errorCantKickBanSelf)
		return
	}

	var valid bool
	row := db.QueryRow("SELECT EXISTS (SELECT * FROM guilds WHERE id=$1 AND owner_id=$2)", ban.Guild, user.Id)
	if row.Err() != nil {
		reportError(http.StatusInternalServerError, w, row.Err())
		return
	}
	row.Scan(&valid)
	if !valid {
		reportError(http.StatusBadRequest, w, errorNotGuildOwner)
		return
	}
	_, err = db.Exec("UPDATE userguilds SET banned=true WHERE guild_id=$1 AND user_id=$2", ban.Guild, ban.Id)
	if err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	broadcastClient(ban.Id, leaveGuildData{Guild: ban.Guild})
	removeUserFromPool(ban.Guild, ban.Id)
	broadcastGuild(ban.Guild, userGuildRemove{Guild: ban.Guild, Id: ban.Id})

	row = db.QueryRow("SELECT username, id FROM users WHERE id=$1", ban.Id)
	var userban userBannedAdd
	if err := row.Scan(&userban.User.Username, &userban.User.Id); err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	userban.Guild = ban.Guild
	userban.User.Icon = 0             //temp
	broadcastClient(user.Id, userban) //update banned user list
	w.WriteHeader(http.StatusOK)
}

func getBannedList(w http.ResponseWriter, r *http.Request, user *session) {
	params := r.URL.Query()
	guild, err := strconv.Atoi(params.Get("guild"))
	if err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}
	row := db.QueryRow("SELECT EXISTS (SELECT * FROM guilds WHERE id=$1 AND owner_id=$2)", guild, user.Id)

	var valid bool

	if err := row.Scan(&valid); err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	if !valid {
		reportError(http.StatusBadRequest, w, errorNotGuildOwner)
		return
	}

	rows, err := db.Query(
		`
		SELECT u.id, u.username
		FROM userguilds g INNER JOIN users u ON u.id = g.user_id
		WHERE g.banned = true AND g.guild_id = $1`,
		guild,
	)
	if err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	userlist := []userGuild{}
	for rows.Next() {
		var user userGuild
		user.Icon = 0 //temp
		if err := rows.Scan(&user.Id, &user.Username); err != nil {
			reportError(http.StatusInternalServerError, w, err)
			return
		}
		userlist = append(userlist, user)
	}
	bodyBytes, err := json.Marshal(userlist)
	if err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(bodyBytes)
}

func unbanUser(w http.ResponseWriter, r *http.Request, user *session) {
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}
	var unban unbanUserData
	err = json.Unmarshal(bodyBytes, &unban)
	if err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}

	row := db.QueryRow("SELECT EXISTS (SELECT * FROM guilds WHERE id=$1 AND owner_id=$2)", unban.Guild, user.Id)

	var valid bool

	if err := row.Scan(&valid); err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	if !valid {
		reportError(http.StatusBadRequest, w, errorNotGuildOwner)
		return
	}

	_, err = db.Exec("DELETE FROM userguilds WHERE guild_id=$1 AND user_id=$2", unban.Guild, unban.Id)
	if err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	broadcastClient(user.Id, userBannedRemove{Guild: unban.Guild, Id: unban.Id})
	w.WriteHeader(http.StatusOK)
}

func leaveGuild(w http.ResponseWriter, r *http.Request, user *session) {
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}
	var leave leaveGuildData
	err = json.Unmarshal(bodyBytes, &leave)
	if err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}

	row := db.QueryRow("SELECT EXISTS (SELECT * FROM guilds WHERE id=$1 AND owner_id=$2)", leave.Guild, user.Id)

	var valid bool

	if err := row.Scan(&valid); err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	if valid {
		reportError(http.StatusBadRequest, w, errorCantLeaveOwnGuild)
		return
	}

	row = db.QueryRow("SELECT EXISTS(SELECT * FROM userguilds WHERE guild_id=$1 AND user_id=$2 AND banned = false)", leave.Guild, user.Id)
	if err := row.Scan(&valid); err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	if !valid {
		reportError(http.StatusBadRequest, w, errorNotInGuild)
		return
	}
	_, err = db.Exec("DELETE FROM userguilds WHERE guild_id=$1 AND user_id=$2", leave.Guild, user.Id)
	if err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	broadcastClient(user.Id, leaveGuildData{Guild: leave.Guild})
	removeUserFromPool(leave.Guild, user.Id)
	broadcastGuild(leave.Guild, userGuildRemove{Guild: leave.Guild, Id: user.Id})
	w.WriteHeader(http.StatusOK)
}

func getGuildSettings(w http.ResponseWriter, r *http.Request, user *session) {
	params := r.URL.Query()
	guild, err := strconv.Atoi(params.Get("guild"))
	if err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}

	var valid bool
	row := db.QueryRow("SELECT EXISTS (SELECT * FROM guilds WHERE owner_id=$1 AND id=$2)", user.Id, guild)
	if err := row.Scan(&valid); err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	if !valid {
		reportError(http.StatusBadRequest, w, errorNotGuildOwner)
		return
	}
	row = db.QueryRow("SELECT save_chat FROM guilds WHERE id=$1", guild)
	var guildData getGuildSettingData
	if err := row.Scan(&guildData.SaveChat); err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	bodyBytes, err := json.Marshal(guildData)
	if err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(bodyBytes)
}
