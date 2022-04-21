package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/asianchinaboi/backendserver/logger"
)

/*
Guilds table
id PRIMARY KEY SERIAL | name VARCHAR(16) | icon INT
*/
/*
Invites table
invite VARCHAR(10) | guild_id INT
*/
type reqCreateGuild struct {
	Icon int // if icon none its zero assume no icon
	Name string
}

type reqInvite struct {
	Invite string
	Guild  int
}

func createGuild(w http.ResponseWriter, r *http.Request) {
	var guild reqCreateGuild
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.WriteLog(logger.ERROR, err.Error())
		w.Write([]byte("bad request"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err := json.Unmarshal(bodyBytes, &guild); err != nil {
		log.WriteLog(logger.ERROR, err.Error())
		w.Write([]byte("bad request"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if _, err := db.Exec("INSERT INTO guilds (name, icon) VALUES ($1, $2)", guild.Name, guild.Icon); err != nil {
		log.WriteLog(logger.ERROR, err.Error())
		w.Write([]byte("bad request"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	invite := reqInvite{
		Invite: generateRandString(10),
		Guild:  0, //find sql id later
	}
	if _, err := db.QueryRow("INSERT INTO invites (invite, guild_id) VALUES ($1, $2) RETURNING id", invite.Invite, invite.Guild); err != nil {
		log.WriteLog(logger.ERROR, err.Error())
		w.Write([]byte("bad request"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	bodyBytes, err = json.Marshal(invite)
	if err != nil {
		log.WriteLog(logger.ERROR, err.Error())
		w.Write([]byte("bad request"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.Write(bodyBytes)
	w.WriteHeader(http.StatusOK)
}
