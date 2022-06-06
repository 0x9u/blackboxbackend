package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
)

/*
Guilds table
id PRIMARY KEY SERIAL | name VARCHAR(16) | icon INT | owner_id INT | invites_amount SMALLINT
*/
/*
Invites table
invite VARCHAR(10) | guild_id INT | ExpireDate (IMPLEMENT THIS LATER ON) INT64
*/
type reqCreateGuild struct {
	Icon int    `json:"Icon"` // if icon none its zero assume no icon
	Name string `json:"Name"`
}

type getInvite struct {
	Guild int `json:"Guild"`
}

type deleteInvite struct {
	Guild  int    `json:"Guild"`
	Invite string `json:"invite"`
}

type reqInvite struct {
	Invite string `json:"Invite"`
	Guild  int    `json:"Guild"`
}

func createGuild(w http.ResponseWriter, r *http.Request) {

	token, ok := r.Header["Auth-Token"]
	if !ok || len(token) == 0 {
		reportError(http.StatusBadRequest, w, errorToken)
		return
	}
	user, err := checkToken(token[0])
	if err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}

	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}
	var guild reqCreateGuild
	if err := json.Unmarshal(bodyBytes, &guild); err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}
	var guild_id int
	row := db.QueryRow("INSERT INTO guilds (name, icon, owner_id) VALUES ($1, $2, $3) RETURNING id", guild.Name, guild.Icon, user.Id)
	row.Scan(&guild_id)
	invite := reqInvite{
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

func genGuildInvite(w http.ResponseWriter, r *http.Request) {

	token, ok := r.Header["Auth-Token"]
	if !ok || len(token) == 0 {
		reportError(http.StatusBadRequest, w, errorToken)
		return
	}
	_, err := checkToken(token[0])
	if err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}

	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}
	var inv getInvite
	if err := json.Unmarshal(bodyBytes, &inv); err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}
	invite := reqInvite{
		Invite: generateRandString(10),
		Guild:  inv.Guild,
	}
	if _, err := db.Exec("INSERT INTO invites (invite, guild_id) VALUES ($1, $2)", invite.Invite, invite.Guild); err != nil {
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

func deleteInvGuild(w http.ResponseWriter, r *http.Request) {
	token, ok := r.Header["Auth-Token"]
	if !ok || len(token) == 0 {
		reportError(http.StatusBadRequest, w, errorToken)
		return
	}
	_, err := checkToken(token[0])
	if err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}
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
