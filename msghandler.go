package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/asianchinaboi/backendserver/logger"
)

type msg struct {
	Id      int    `json:"Id"`
	Author  int    `json:"Author"`  // author id aka user id
	Content string `json:"Content"` // message content
	Guild   int    `json:"Guild"`   // Chat id
	Time    int64  `json:"Time"`
}

/*
type content struct {
	Id     int `json:"Id"`
	Author int `json`
}
*/

func msgRecieve(w http.ResponseWriter, r *http.Request) {

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
	var datamsg msg
	log.WriteLog(logger.INFO, string(bodyBytes))
	err = json.Unmarshal(bodyBytes, &datamsg)
	datamsg.Time = time.Now().Unix()
	if err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}
	log.WriteLog(logger.INFO, fmt.Sprintf("Message recieved %s", datamsg.Content))

	//send msg to database
	//broadcast msg to all connections to websocket
	var id int
	row := db.QueryRow("SELECT user_id FROM userguilds guild_id=$1 AND user_id=$2")
	row.Scan(&id)
	if id != user.Id {
		reportError(http.StatusBadRequest, w, errorNotInGuild)
		return
	}

	if _, err = db.Exec("INSERT INTO messages (content, user_id, guild_id, time) VALUES ($1, $2, $3, $4)", datamsg.Content, user.Id, datamsg.Guild, datamsg.Time); err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}

	rows, err := db.Query("SELECT user_id FROM userguilds WHERE guild_id=$1", datamsg.Guild)
	if err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}
	defer rows.Close()
	var ids []int
	for rows.Next() {
		var id int
		err := rows.Scan(&id)
		if err != nil {
			reportError(http.StatusInternalServerError, w, err)
			return
		}
		ids = append(ids, id)
	}
	for id := range ids {
		client := clients[id]
		if client == nil {
			continue
		}
		client <- datamsg
	}
	w.WriteHeader(http.StatusOK)
}

func msgSend(w http.ResponseWriter, r *http.Request) {

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

	messages := []msg{}

	limit := r.URL.Query().Get("limit")
	timestamp := r.URL.Query().Get("time")

	if timestamp == "" {
		timestamp = fmt.Sprintf("%v", time.Now().Unix())
	}

	if limit == "" {
		limit = "50"
	}

	log.WriteLog(logger.INFO, fmt.Sprintf("limit: %v, timestamp %v", limit, timestamp))
	rows, err := db.Query("SELECT * FROM messages WHERE time <= $1 ORDER BY time DESC LIMIT $2", timestamp, limit)
	if err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		message := msg{}
		err := rows.Scan(&message.Id, &message.Content, &message.Author, &message.Guild, &message.Time)
		if err != nil {
			reportError(http.StatusInternalServerError, w, err)
			return
		}
		messages = append(messages, message)
	}
	result, err := json.Marshal(messages)
	if err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(result)
}
