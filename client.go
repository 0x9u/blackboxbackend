package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/asianchinaboi/backendserver/logger"
	"github.com/gorilla/websocket"
)

type brcastEvents chan interface{}

type client struct {
	ws        *websocket.Conn
	id        int
	guilds    []int //not used might remove later
	timer     *time.Ticker
	broadcast brcastEvents
	quit      context.Context //chan bool //also temporary maybe use contexts later
	quitFunc  context.CancelFunc
	//	keepAlive bool //temporary try find solution
}

type sendDataType struct {
	DataType int         //0 for pingpong 1 for message 2 for guild Changes
	Data     interface{} //data to send
}

type pingpong struct {
	Data string //probably include more stuff maybe idk
}

var lock sync.Mutex

const (
	pingDelay    = 10 * time.Second
	messageLimit = 2 << 7 //256
	pongDelay    = (pingDelay * 2) / 5
)

func (c *client) run() {
	defer func() {
		err := c.ws.Close()
		if err != nil {
			log.WriteLog(logger.ERROR, fmt.Errorf("an error occured when leaving websocket %v", err).Error())
		}
		close(clients[c.id])
		delete(clients, c.id)
		log.WriteLog(logger.INFO, "Websocket of "+c.ws.LocalAddr().String()+" has been closed")
	}()
	log.WriteLog(logger.INFO, "Websocket active of "+c.ws.LocalAddr().String())
	go c.heartBeat()
	c.ws.SetReadLimit(messageLimit)
	for {
		select {
		//recieve messages
		case data, ok := <-clients[c.id]:
			if !ok {
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
			log.WriteLog(logger.INFO, "Writing to client")
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
		var recieved pingpong
		err = json.Unmarshal(message, &recieved)
		if err != nil {
			log.WriteLog(logger.WARN, "an error occured during unmarshalling with websocket: "+c.ws.LocalAddr().String()+": "+err.Error())
			continue
		}
		c.ws.SetReadDeadline(time.Now().Add(pingDelay)) //screw handlers
		c.timer.Reset(pongDelay)                        //just in case if client pings in multiple intervals

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
	}
	sendData := sendDataType{
		DataType: dataType,
		Data:     data,
	}
	err := c.ws.WriteJSON(sendData)
	if err != nil {
		log.WriteLog(logger.ERROR, err.Error())
		//c.quit <- true
		c.quitFunc()
	}
}

func webSocket(w http.ResponseWriter, r *http.Request, user *session) {
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
	for rows.Next() {
		var guild int
		rows.Scan(&guild)
		guilds = append(guilds, guild)
		lock.Lock() //slow needs fix
		//only implementing to prevent data races
		guildPool, ok := pools[guild]
		clientData := addClientData{
			Id: user.Id,
			Ch: broadcastChannel,
		}
		if !ok {
			createPool(guild)
			pools[guild].Add <- clientData
		} else {
			guildPool.Add <- clientData
		}
		lock.Unlock()
		fmt.Println("successful")
	}
	//get all the guilds the user is in
	rows.Close()
	ctx := context.Background()
	quit, quitFunc := context.WithCancel(ctx)
	instanceuser := client{
		ws:        ws,
		id:        user.Id, //broadcast to all clients
		guilds:    guilds,
		timer:     time.NewTicker(pongDelay),
		broadcast: broadcastChannel,
		quit:      quit,
		quitFunc:  quitFunc,
	}
	clients[user.Id] = instanceuser.broadcast
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
	client := clients[id]
	if client == nil {
		return http.StatusBadRequest, fmt.Errorf("client %d does not exist", id)
	}
	client <- data
	return 0, nil
}
