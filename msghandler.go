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
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("bad request"))
		log.WriteLog(logger.ERROR, err.Error())
		return
	}
	var datamsg msg
	log.WriteLog(logger.INFO, string(bodyBytes))
	err = json.Unmarshal(bodyBytes, &datamsg)
	datamsg.Time = time.Now().Unix()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("bad request"))
		log.WriteLog(logger.ERROR, err.Error())
		return
	}
	log.WriteLog(logger.INFO, fmt.Sprintf("Message recieved %s", datamsg.Content))

	//send msg to database
	//broadcast msg to all connections to websocket

	_, err = db.Exec("INSERT INTO messages (content, author_id, guild_id, time) VALUES ($1, $2, $3, $4)", datamsg.Content, datamsg.Author, datamsg.Guild, datamsg.Time)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.WriteLog(logger.ERROR, err.Error())
		return
	}

	rows, err := db.Query("SELECT user_id FROM userguilds WHERE guild_id=$1", datamsg.Guild)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.WriteLog(logger.ERROR, err.Error())
		return
	}
	defer rows.Close()
	var ids []int
	for rows.Next() {
		var id int
		err := rows.Scan(&id)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("bad request"))
			log.WriteLog(logger.ERROR, err.Error())
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
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("bad request"))
		log.WriteLog(logger.ERROR, err.Error())
		return
	}
	defer rows.Close()
	for rows.Next() {
		message := msg{}
		err := rows.Scan(&message.Id, &message.Content, &message.Author, &message.Guild, &message.Time)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("bad request"))
			log.WriteLog(logger.ERROR, err.Error())
			return
		}
		messages = append(messages, message)
	}
	result, err := json.Marshal(messages)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("bad request"))
		log.WriteLog(logger.ERROR, err.Error())
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(result)
}

func msgSocket(w http.ResponseWriter, r *http.Request) {

	token := r.Header.Get("token")
	if token == "" {
		log.WriteLog(logger.ERROR, "Token is not provided")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.WriteLog(logger.ERROR, err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	//defer ws.Close()
	id := tokens[token].Id
	if id == 0 {
		log.WriteLog(logger.INFO, fmt.Sprintf("Invalid token: %v", token))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	rows, err := db.Query("SELECT guild_id FROM userguilds WHERE user_id=$1", id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("bad request"))
		log.WriteLog(logger.ERROR, err.Error())
		return
	}
	var guilds []int
	for rows.Next() {
		var guild int
		rows.Scan(&guild)
		guilds = append(guilds, guild)
	}
	rows.Close()
	clients[id] = make(brcastEvents)
	user := client{
		ws:          ws,
		id:          id,
		guilds:      guilds,
		broadcaster: clients[id],
	}
	user.run()

}
