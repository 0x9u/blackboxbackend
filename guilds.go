package main

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
)

/*
Guilds table
id PRIMARY KEY SERIAL | name VARCHAR(16) | icon INT | owner_id INT | invites_amount SMALLINT
*/
/*
Invites table
invite VARCHAR(10) | guild_id INT | ExpireDate (IMPLEMENT THIS LATER ON) INT64
*/
type createGuildData struct {
	Icon int    `json:"Icon"` // if icon none its zero assume no icon
	Name string `json:"Name"`
}

type deleteGuildData struct {
	Guild int `json:"Guild"`
}

type deleteInvite struct {
	Guild  int    `json:"Guild"`
	Invite string `json:"invite"`
}

type sendInvite struct {
	Invite string `json:"Invite"`
	Guild  int    `json:"Guild"`
}

func createGuild(w http.ResponseWriter, r *http.Request, user *session) {

	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}
	var guild createGuildData
	if err := json.Unmarshal(bodyBytes, &guild); err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}
	var guild_id int
	row := db.QueryRow("INSERT INTO guilds (name, icon, owner_id) VALUES ($1, $2, $3) RETURNING id", guild.Name, guild.Icon, user.Id)
	row.Scan(&guild_id)
	invite := sendInvite{
		Invite: generateRandString(10),
		Guild:  guild_id,
	}
	if _, err = db.Exec("INSERT INTO userguilds (guild_id, user_id) VALUES ($1, $2)", guild_id, user.Id); err != nil { //cleanup if failed later
		reportError(http.StatusBadRequest, w, err)
		return
	}
	if _, err = db.Exec("INSERT INTO invites (invite, guild_id) VALUES ($1, $2)", invite.Invite, invite.Guild); err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}
	bodyBytes, err = json.Marshal(invite)
	if err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(bodyBytes)
}

func deleteGuild(w http.ResponseWriter, r *http.Request, user *session) {
	guildParam := r.URL.Query().Get("Guild")
	guild, err := strconv.Atoi(guildParam)
	if err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}
	row := db.QueryRow("SELECT id FROM guilds WHERE id=$1 AND owner_id=$2", guild, user.Id)
	if err := row.Err(); err == sql.ErrNoRows {
		reportError(http.StatusBadRequest, w, err)
		return
	} else if err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	_, err = db.Exec("DELETE FROM guilds WHERE id=$1", guild)
	if err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	_, err = db.Exec("DELETE FROM userguilds WHERE guild_id=$1", guild)
	if err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	_, err = db.Exec("DELETE FROM invites WHERE guild_id=$1", guild)
	if err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	_, err = db.Exec("DELETE FROM messages WHERE guild_id=$1", guild)
	if err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func genGuildInvite(w http.ResponseWriter, r *http.Request, user *session) {
	guildParam := r.URL.Query().Get("guild")
	guild, err := strconv.Atoi(guildParam)
	if err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}
	invite := sendInvite{
		Invite: generateRandString(10),
		Guild:  guild,
	}
	if _, err := db.Exec("INSERT INTO invites (invite, guild_id) VALUES ($1, $2)", invite.Invite, invite.Guild); err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}
	bodyBytes, err := json.Marshal(invite)
	if err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	w.Write(bodyBytes)
	w.WriteHeader(http.StatusOK)
}

func deleteInvGuild(w http.ResponseWriter, r *http.Request, user *session) {
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}
	var inv deleteInvite
	err = json.Unmarshal(bodyBytes, &inv)
	if err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}
	row := db.QueryRow("DELETE FROM guilds WHERE guild_id=$1, invite=$2", inv.Guild, inv.Invite)
	if row.Err() != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}
