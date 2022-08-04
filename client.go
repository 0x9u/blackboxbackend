package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/asianchinaboi/backendserver/logger"
	"github.com/gorilla/websocket"
)

type brcastEvents chan interface{}

type client struct {
	ws        *websocket.Conn
	id        int
	uniqueId  string //since some guys might be using multiple connections on one account
	guilds    []int  //not used might remove later
	timer     *time.Ticker
	broadcast brcastEvents
	quit      context.Context //chan bool //also temporary maybe use contexts later
	quitFunc  context.CancelFunc
	//	keepAlive bool //temporary try find solution
}

type sendDataType struct {
	DataType int         `json:"dataType"` //0 for pingpong 1 for message 2 for guild Changes
	Data     interface{} `json:"data"`     //data to send
}

type pingpong struct {
	Data string //probably include more stuff maybe idk
}

var (
	lockPool  sync.Mutex // to be replaced with mutex map
	lockAlias sync.Mutex //to be replace with mutex map
	//slow ass code yuck
)

const (
	pingDelay    = 20 * time.Second
	messageLimit = 2 << 7 //256
	pongDelay    = (pingDelay * 2) / 5
)

func (c *client) run() {
	defer func() {
		err := c.ws.Close()
		if err != nil {
			log.WriteLog(logger.ERROR, fmt.Errorf("an error occured when leaving websocket %v", err).Error())
		}
		close(clients[c.uniqueId]) //close of nil channel error occurs here sometimes
		delete(clients, c.uniqueId)
		delete(clientAlias[c.id], c.uniqueId)

		lockAlias.Lock()
		if len(clientAlias[c.id]) == 0 {
			delete(clientAlias, c.id)
		}
		lockAlias.Unlock()

		log.WriteLog(logger.INFO, "Websocket of "+c.ws.LocalAddr().String()+" has been closed")
		//leaves the guild pools
		rows, err := db.Query("SELECT guild_id FROM userguilds WHERE user_id = $1 AND banned = false", c.id)
		if err != nil {
			log.WriteLog(logger.ERROR, fmt.Errorf("an error occured when getting guilds of user %v", err).Error())
			return
		}
		for rows.Next() { //should be using guilds array instead lol
			var guildId int
			err = rows.Scan(&guildId)
			if err != nil {
				log.WriteLog(logger.ERROR, fmt.Errorf("an error occured when getting guilds of user %v", err).Error())
				return
			}
			_, ok := pools[guildId]
			if !ok { // shouldnt happen but just in case
				continue
			}
			lockPool.Lock()
			pools[guildId].Remove <- c.uniqueId //RACE CONDITION SEE pool.go:30
			lockPool.Unlock()
		}
	}()
	log.WriteLog(logger.INFO, "Websocket active of "+c.ws.LocalAddr().String())
	go c.heartBeat()
	c.ws.SetReadLimit(messageLimit)
	for {
		select {
		//recieve messages
		case data, ok := <-c.broadcast:
			if !ok { //idk if this is needed or not
				c.quitFunc() //call cancel but never actually recieve it
				return       //the channel has been closed get out of here
			}
			c.eventCheck(data)
		case <-c.quit.Done(): //<-c.quit:
			return
		case <-c.timer.C:
			data := sendDataType{
				DataType: 0,
				Data:     pingpong{Data: "pong"},
			}
			message, _ := json.Marshal(data)
			//log.WriteLog(logger.INFO, "Writing to client")
			if err := c.ws.WriteMessage(websocket.TextMessage, message); err != nil {
				c.quitFunc()
			}
		}
	}
}

func (c *client) heartBeat() {
	c.ws.SetReadDeadline(time.Now().Add(pingDelay)) //note to self put that thing in seconds otherwise its goddamn miliseconds which is hard to debug
	for {                                           //need to check for quit
		_, message, err := c.ws.ReadMessage()
		if err != nil { //should usually return io error which is fine since it means the websocket has timeouted
			log.WriteLog(logger.ERROR, err.Error()) //or if the websocket has closed which is a 1000 (normal)
			//c.quit <- true
			c.quitFunc() //if recieve websocket closed error then you gotta do what you gotta do
			log.WriteLog(logger.INFO, "Disconnecting websocket")
			break
		}
		//		log.WriteLog(logger.INFO, "string json:"+string(message))
		var recieved sendDataType
		err = json.Unmarshal(message, &recieved)
		if err != nil {
			log.WriteLog(logger.WARN, "an error occured during unmarshalling with websocket: "+c.ws.LocalAddr().String()+": "+err.Error())
			continue
		}
		//		log.WriteLog(logger.INFO, "Recieved info")
		if recieved.DataType == 0 {
			//			log.WriteLog(logger.INFO, "Has been pinged")
			c.ws.SetReadDeadline(time.Now().Add(pingDelay)) //screw handlers
			c.timer.Reset(pongDelay)                        //just in case if client pings in multiple intervals
		}
	}
}

func (c *client) eventCheck(data interface{}) {
	var dataType int
	switch data.(type) {
	case msg:
		dataType = 1
	case deleteMsg:
		dataType = 2
	case editMsg:
		dataType = 3
	case changeGuild: //implement files soon or something idk guild change ban or kick whatever
		dataType = 4
	case joinGuildData: //join guild from user
		dataType = 5
	case leaveGuildData: //user banned/kicked
		dataType = 6
	case userGuildAdd: // Updates the user List (APPEND)
		dataType = 7
	case userGuildRemove:
		dataType = 8
	case userBannedAdd:
		dataType = 9
	case userBannedRemove:
		dataType = 10
	case inviteAdded:
		dataType = 11
	case inviteRemoved:
		dataType = 12
	default:
		log.WriteLog(logger.WARN, fmt.Sprintf("Invalid data type recieved: %v", data))
		return
	}
	sendData := sendDataType{
		DataType: dataType,
		Data:     data,
	}
	log.WriteLog(logger.INFO, "Sending info of type "+strconv.Itoa(dataType))
	err := c.ws.WriteJSON(sendData)
	if err != nil {
		log.WriteLog(logger.ERROR, err.Error())
		//c.quit <- true
		c.quitFunc()
	}
}

func webSocket(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if len(token) == 0 {
		reportError(http.StatusBadRequest, w, errorToken)
		return
	}
	user, err := checkToken(token)
	if err != nil {
		reportError(http.StatusBadRequest, w, err)
		return
	}

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}

	rows, err := db.Query("SELECT guild_id FROM userguilds WHERE user_id=$1", user.Id)
	if err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	var guilds []int
	broadcastChannel := make(brcastEvents)
	uniqueId := generateRandString(32)

	for rows.Next() {
		var guild int
		rows.Scan(&guild)
		guilds = append(guilds, guild)
		lockPool.Lock() //slow needs fix
		//only implementing to prevent data races
		guildPool, ok := pools[guild]
		clientData := addClientData{
			UniqueId: uniqueId,
			Ch:       broadcastChannel,
		}
		if !ok {
			createPool(guild)
			pools[guild].Add <- clientData
		} else {
			guildPool.Add <- clientData
		}
		lockPool.Unlock()
	}
	//get all the guilds the user is in
	rows.Close()
	ctx := context.Background()
	quit, quitFunc := context.WithCancel(ctx)
	instanceuser := client{
		ws:        ws,
		id:        user.Id, //broadcast to all clients
		uniqueId:  uniqueId,
		guilds:    guilds,
		timer:     time.NewTicker(pongDelay),
		broadcast: broadcastChannel,
		quit:      quit,
		quitFunc:  quitFunc,
	}
	clients[uniqueId] = instanceuser.broadcast
	lockAlias.Lock() //prevents datarace
	if _, ok := clientAlias[user.Id]; !ok {
		clientAlias[user.Id] = make(map[string]brcastEvents)
	}
	lockAlias.Unlock()
	clientAlias[user.Id][uniqueId] = instanceuser.broadcast
	instanceuser.run()

}

/*
func broadcastGuild(guild int, data interface{}) (statusCode int, err error) {
	rows, err := db.Query("SELECT user_id FROM userguilds WHERE guild_id=$1", guild)
	if err != nil {
		//reportError(http.StatusBadRequest, w, err)
		return http.StatusBadRequest, err
	}
	defer rows.Close()
	var ids []int
	for rows.Next() {
		var id int
		err := rows.Scan(&id)
		if err != nil {
			//reportError(http.StatusInternalServerError, w, err)
			return http.StatusInternalServerError, err
		}
		ids = append(ids, id)
	}
	for _, id := range ids {
		client := clients[id]
		log.WriteLog(logger.INFO, fmt.Sprintf("clientslist %v", data))
		if client == nil {
			continue
		}
		client <- data
		log.WriteLog(logger.INFO, fmt.Sprintf("Message sent to %d", id))
	}
	return 0, nil
}
*/

func broadcastClient(id int, data interface{}) (statusCode int, err error) {
	lockAlias.Lock()
	clientList, ok := clientAlias[id] //problem if two websockets of same user exist only of those two will be sent
	if !ok {
		return http.StatusBadRequest, fmt.Errorf("client %d does not exist", id)
	}
	for _, client := range clientList {
		client <- data
	}
	lockAlias.Unlock()
	return 0, nil
}
