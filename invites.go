package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
)

func genGuildInvite(w http.ResponseWriter, r *http.Request, user *session) {
	bytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}
	var guildInfo sendInvite
	err = json.Unmarshal(bytes, &guildInfo)
	if err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}
	guild := guildInfo.Guild
	invite := sendInvite{
		Invite: generateRandString(10),
		Guild:  guild,
	}
	row := db.QueryRow("SELECT COUNT(*) FROM invites WHERE guild_id=$1", guild)
	var count int
	if err := row.Scan(&count); err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	if count > 9 {
		reportError(http.StatusBadRequest, w, errorInviteLimitReached)
		return
	}
	if _, err := db.Exec("INSERT INTO invites (invite, guild_id) VALUES ($1, $2)", invite.Invite, invite.Guild); err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	bodyBytes, err := json.Marshal(invite)
	if err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(bodyBytes)
}

func deleteInvGuild(w http.ResponseWriter, r *http.Request, user *session) {
	params := r.URL.Query()
	invite := params.Get("invite")
	guild, err := strconv.Atoi(params.Get("guild"))
	if err != nil || invite == "" {
		reportError(http.StatusBadRequest, w, err)
		return
	}
	var valid bool
	row := db.QueryRow("SELECT EXISTS (SELECT * FROM guilds WHERE id=$1 AND owner_id=$2)", guild, user.Id)
	if row.Err() != nil {
		reportError(http.StatusInternalServerError, w, row.Err())
		return
	}
	row.Scan(&valid)
	if !valid {
		reportError(http.StatusBadRequest, w, errorNotGuildOwner)
		return
	}
	_, err = db.Exec("DELETE FROM guilds WHERE guild_id=$1 AND invite=$2", guild, invite)
	if err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func getGuildInvite(w http.ResponseWriter, r *http.Request, user *session) {
	params := r.URL.Query()
	guild, err := strconv.Atoi(params.Get("guild"))
	if err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}
	row := db.QueryRow("SELECT EXISTS (SELECT * FROM userguilds WHERE guild_id=$1 AND user_id=$2)", guild, user.Id)
	if row.Err() != nil {
		reportError(http.StatusInternalServerError, w, row.Err())
		return
	}
	var valid bool
	row.Scan(&valid)
	if !valid {
		reportError(http.StatusBadRequest, w, errorNotInGuild)
		return
	}
	invite := sendInvite{
		Guild: guild,
	}
	row = db.QueryRow("SELECT invite FROM invites WHERE guild_id=$1 LIMIT 1", guild)
	if row.Err() != nil {
		reportError(http.StatusInternalServerError, w, row.Err())
		return
	}
	row.Scan(&invite.Invite)
	bodyBytes, err := json.Marshal(invite)
	if err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(bodyBytes)
}
