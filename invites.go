package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/asianchinaboi/backendserver/logger"
)

type inviteUploadData struct {
	Invite string `json:"Invite"`
	Guild  int    `json:"Guild"` // probably unused remove later
}

type inviteAdded inviteUploadData
type inviteRemoved inviteUploadData

func genGuildInvite(w http.ResponseWriter, r *http.Request, user *session) {
	bytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}
	var guildInfo inviteUploadData
	err = json.Unmarshal(bytes, &guildInfo)
	if err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}
	invite := inviteAdded{
		Invite: generateRandString(10),
		Guild:  guildInfo.Guild,
	}
	row := db.QueryRow("SELECT COUNT(*) FROM invites WHERE guild_id=$1", guildInfo.Guild)
	var count int
	if err := row.Scan(&count); err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	if count > 9 {
		reportError(http.StatusBadRequest, w, errorInviteLimitReached)
		return
	}
	if _, err := db.Exec("INSERT INTO invites (invite, guild_id) VALUES ($1, $2)", invite.Invite, guildInfo.Guild); err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	bodyBytes, err := json.Marshal(invite)
	if err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	broadcastGuild(guildInfo.Guild, invite)
	w.WriteHeader(http.StatusOK)
	w.Write(bodyBytes)
}

func deleteInvGuild(w http.ResponseWriter, r *http.Request, user *session) {
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}
	var invite inviteUploadData
	err = json.Unmarshal(bodyBytes, &invite)
	if err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}
	if invite.Invite == "" {
		reportError(http.StatusBadRequest, w, errorNoInvite)
		return
	}
	var valid bool
	row := db.QueryRow("SELECT EXISTS (SELECT * FROM guilds WHERE id=$1 AND owner_id=$2)", invite.Guild, user.Id)
	if err := row.Scan(&valid); err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	if !valid {
		reportError(http.StatusBadRequest, w, errorNotGuildOwner)
		return
	}
	row = db.QueryRow("SELECT EXISTS (SELECT * FROM invites WHERE guild_id=$1 AND invite = $2)", invite.Guild, invite.Invite)
	if err := row.Scan(&valid); err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	if !valid {
		reportError(http.StatusBadRequest, w, errorInvalidInvite)
		return
	}
	_, err = db.Exec("DELETE FROM invites WHERE guild_id=$1 AND invite=$2", invite.Guild, invite.Invite)
	if err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	log.WriteLog(logger.INFO, "before broadcast")
	broadcastGuild(invite.Guild, inviteRemoved(invite))
	log.WriteLog(logger.INFO, "after broadcast")
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
	rows, err := db.Query("SELECT invite FROM invites WHERE guild_id=$1", guild)
	if err != nil {
		reportError(http.StatusInternalServerError, w, row.Err())
		return
	}
	inviteList := []string{}
	for rows.Next() {
		var invite string
		if err := rows.Scan(&invite); err != nil {
			reportError(http.StatusInternalServerError, w, err)
			return
		}
		inviteList = append(inviteList, invite)
	}
	bodyBytes, err := json.Marshal(inviteList)
	if err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(bodyBytes)
}
