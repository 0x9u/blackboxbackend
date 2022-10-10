package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/asianchinaboi/backendserver/logger"
)

type msg struct {
	Id        int    `json:"Id"`
	Author    author `json:"Author"`  // author id aka user id
	Content   string `json:"Content"` // message content
	Guild     int    `json:"Guild"`   // Chat id
	Time      int64  `json:"Time"`
	MsgSaved  bool   `json:"MsgSaved"`  //shows if the message is saved or not
	Edited    bool   `json:"Edited"`    //shows if msg has been edited
	RequestId string `json:"RequestId"` //only important for the sender (NOW IMPORTANT IF CHAT HISTORY IS OFF IT IS A REPLACEMENT FOR THE ID)
}

type author struct {
	Id       int    `json:"Id"`
	Username string `json:"Username"`
	Icon     int    `json:"Icon"` //will be implemented later
}

type deleteMsg struct {
	Id        int    `json:"Id"`
	RequestID string `json:"RequestId"` //only used if chat history is not on it is a replacement for Id
	Author    int    `json:"Author"`    //delete all messages from author
	Guild     int    `json:"Guild"`     //delete all messages from guild
	Time      int    `json:"Time"`      //delete messages up to timestamp
}

type clearUserMsgData struct {
	Id    int `json:"Id"`
	Guild int `json:"Guild"`
}

type clearGuildMsgData struct {
	Guild int `json:"Guild"`
}

type editMsg struct {
	Id        int    `json:"Id"`        //msg id
	RequestID string `json:"RequestId"` //only used if chat history is not on it is a replacement for Id
	Guild     int    `json:"Guild"`
	Content   string `json:"Content"` //new msg content
}

/*
type content struct {
	Id     int `json:"Id"`
	Author int `json`
}
*/

func msgRecieve(w http.ResponseWriter, r *http.Request, user *session) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}
	var datamsg msg
	log.WriteLog(logger.INFO, string(bodyBytes))
	err = json.Unmarshal(bodyBytes, &datamsg)
	datamsg.Time = time.Now().UnixMilli()
	if err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}
	log.WriteLog(logger.INFO, fmt.Sprintf("Message recieved %s", datamsg.Content))

	//send msg to database
	//broadcast msg to all connections to websocket
	var valid bool
	row := db.QueryRow("SELECT EXISTS (SELECT * FROM userguilds WHERE guild_id=$1 AND user_id=$2 AND banned=false)", datamsg.Guild, user.Id)
	if err := row.Scan(&valid); err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	if !valid {
		reportError(http.StatusBadRequest, w, errorNotInGuild)
		return
	}
	datamsg.Content = strings.TrimSpace(datamsg.Content)
	log.WriteLog(logger.INFO, datamsg.Content)
	if len(datamsg.Content) == 0 {
		reportError(http.StatusBadRequest, w, errorNoMsgContent)
		return
	}

	if len(datamsg.Content) > 1024 {
		reportError(http.StatusBadRequest, w, errorMsgTooLong)
		return
	}

	//check if guild has chat messages save turned on
	row = db.QueryRow("SELECT save_chat FROM guilds WHERE id=$1", datamsg.Guild)
	if err := row.Scan(&valid); err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}

	datamsg.Id = 0 //just there to make it obvious

	if valid {
		row = db.QueryRow("INSERT INTO messages (content, user_id, guild_id, time) VALUES ($1, $2, $3, $4) RETURNING id", datamsg.Content, user.Id, datamsg.Guild, datamsg.Time)
		if err := row.Scan(&datamsg.Id); err != nil {
			reportError(http.StatusBadRequest, w, err)
			return
		}
	}

	datamsg.MsgSaved = valid //false not saved | true saved

	var authorData author
	db.QueryRow("SELECT username FROM users WHERE id=$1", user.Id).Scan(&authorData.Username)
	authorData.Id = user.Id
	authorData.Icon = 0 //placeholder
	datamsg.Author = authorData
	statusCode, err := broadcastGuild(datamsg.Guild, datamsg)
	if err != nil && err != errorGuildPoolNotExist {
		reportError(statusCode, w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func msgHistory(w http.ResponseWriter, r *http.Request, user *session) { //sends message history

	messages := []msg{}

	limit := r.URL.Query().Get("limit")
	timestamp := r.URL.Query().Get("time")
	guild := r.URL.Query().Get("guild")

	if timestamp == "" {
		timestamp = fmt.Sprintf("%v", time.Now().Unix())
	}

	if limit == "" {
		limit = "50"
	}

	if guild == "" {
		reportError(http.StatusBadRequest, w, errorGuildNotProvided)
		return
	}
	var valid bool
	row := db.QueryRow("SELECT EXISTS (SELECT * FROM userguilds WHERE guild_id=$1 AND user_id=$2 AND banned=false)", guild, user.Id)
	if err := row.Err(); err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}
	row.Scan(&valid)
	if !valid {
		reportError(http.StatusBadRequest, w, errorNotInGuild)
		return
	}

	log.WriteLog(logger.INFO, fmt.Sprintf("limit: %v, timestamp %v", limit, timestamp))
	rows, err := db.Query(
		`SELECT m.*, u.username
		FROM messages m INNER JOIN users u 
		ON u.id = m.user_id 
		WHERE time < $1 AND guild_id = $2 
		ORDER BY time DESC LIMIT $3`, //wtf?
		timestamp, guild, limit)
	if err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		message := msg{}
		authorData := author{}
		err := rows.Scan(&message.Id, &message.Content, &authorData.Id,
			&message.Guild, &message.Time, &message.Edited, &authorData.Username)
		authorData.Icon = 0 //placeholder
		message.Author = authorData
		if err != nil {
			reportError(http.StatusInternalServerError, w, err)
			return
		}
		message.MsgSaved = true
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

func msgDelete(w http.ResponseWriter, r *http.Request, user *session) { //deletes message
	var datamsg deleteMsg
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}
	err = json.Unmarshal(bodyBytes, &datamsg)
	if err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}

	if datamsg.Id == 0 && datamsg.Author == 0 && datamsg.RequestID == "" {
		reportError(http.StatusBadRequest, w, errorInvalidDetails)
		return
	} else if datamsg.Guild == 0 {
		reportError(http.StatusBadRequest, w, errorGuildNotProvided)
		return
	}
	var valid bool
	row := db.QueryRow("SELECT EXISTS (SELECT * FROM guilds WHERE id=$1 AND owner_id = $2)", datamsg.Guild, user.Id)
	if row.Err() != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	row.Scan(&valid)
	if !valid && datamsg.Id != 0 && datamsg.Author != user.Id {
		reportError(http.StatusBadRequest, w, errorNotGuildOwner)
		return
	}

	//theortically if the user did not have chat saved and ran this request it would still work
	if datamsg.RequestID == "" {
		row := db.QueryRow("SELECT save_chat from guilds where id = $1", datamsg.Guild)
		row.Scan(&valid)
		if valid { //if save chat is on
			reportError(http.StatusBadRequest, w, errorGuildSaveChatOn)
			return
		}
	} else if datamsg.Id != 0 {
		_, err = db.Exec("DELETE FROM messages where id = $1 AND guild_id = $2 AND user_id = $3", datamsg.Id, datamsg.Guild, datamsg.Author)
	} else { //deprecated make sure it gets its own function too dangerous for api requests
		var deleteMsgUser int
		if datamsg.Author != 0 {
			deleteMsgUser = user.Id
		} else {
			deleteMsgUser = datamsg.Author
		}
		_, err = db.Exec("DELETE FROM messages WHERE time <= $1 AND guild_id = $2 AND user_id = $3", datamsg.Time, datamsg.Guild, deleteMsgUser)
	}
	if err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	statusCode, err := broadcastGuild(datamsg.Guild, datamsg)
	//wont work if time option is picked
	if err != nil {
		reportError(statusCode, w, err)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func msgEdit(w http.ResponseWriter, r *http.Request, user *session) {
	var datamsg editMsg
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}
	err = json.Unmarshal(bodyBytes, &datamsg)
	if err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}
	if (datamsg.Id == 0 && datamsg.RequestID == "") || datamsg.Content == "" || datamsg.Guild == 0 && (datamsg.Id > 0 && len(datamsg.RequestID) > 0) {
		reportError(http.StatusBadRequest, w, errorInvalidDetails)
		return
	}
	if datamsg.RequestID == "" {
		_, err = db.Exec("UPDATE messages SET content = $1, edited = true WHERE id = $2 AND user_id = $3 AND guild_id=$4", datamsg.Content, datamsg.Id, user.Id, datamsg.Guild)
		if err == sql.ErrNoRows {
			reportError(http.StatusBadRequest, w, err)
			return
		} else if err != nil {
			reportError(http.StatusInternalServerError, w, err)
			return
		}
	}
	statusCode, err := broadcastGuild(datamsg.Guild, datamsg)
	if err != nil {
		reportError(statusCode, w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func clearGuildMsg(w http.ResponseWriter, r *http.Request, user *session) {
	var datamsg clearGuildMsgData
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}
	err = json.Unmarshal(bodyBytes, &datamsg)
	if err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}
	if datamsg.Guild == 0 {
		reportError(http.StatusBadRequest, w, errorGuildNotProvided)
		return
	}
	var valid bool
	row := db.QueryRow("SELECT EXISTS (SELECT * FROM guilds WHERE id=$1 AND owner_id = $2)", datamsg.Guild, user.Id)
	if row.Err() != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	row.Scan(&valid)
	if !valid {
		reportError(http.StatusBadRequest, w, errorNotGuildOwner)
		return
	}
	_, err = db.Exec("DELETE FROM messages WHERE guild_id = $1", datamsg.Guild)
	if err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	statusCode, err := broadcastGuild(datamsg.Guild, datamsg)
	if err != nil {
		reportError(statusCode, w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func clearUserMsg(w http.ResponseWriter, r *http.Request, user *session) {
	rows, err := db.Query("SELECT DISTINCT guild_id FROM messages WHERE user_id = $1", user.Id)

	if err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}

	_, err = db.Exec("DELETE FROM messages WHERE user_id = $1", user.Id)
	if err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}

	for rows.Next() {
		var guildId int
		err = rows.Scan(&guildId)
		if err != nil {
			reportError(http.StatusInternalServerError, w, err)
			return
		}
		clearMsg := clearUserMsgData{
			Id:    user.Id,
			Guild: guildId,
		}
		statusCode, err := broadcastGuild(clearMsg.Guild, clearMsg)
		if err != nil && err != errorGuildPoolNotExist {
			reportError(statusCode, w, err)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
}

/*
func msgTyping(w http.ResponseWriter, r *http.Request) {

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
}
*/
