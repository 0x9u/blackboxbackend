package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/asianchinaboi/backendserver/logger"
	"github.com/gorilla/websocket"
)

type brcastEvents chan interface{}

type client struct {
	token    string
	ws       *websocket.Conn
	id       int
	guilds   []int
	timer    *time.Ticker
	quit     context.Context //chan bool //also temporary maybe use contexts later
	quitFunc context.CancelFunc
	//	keepAlive bool //temporary try find solution
}

type ping struct {
	Token string `json:"token"` //probably include more stuff maybe idk
}

type pong struct {
	Token string `json:"token"`
}

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
		//case : //recieve messages
		case data, ok := <-clients[c.id]:
			if !ok {
				c.quitFunc() //call cancel but never actually recieve it
				return       //the channel has been closed get out of here
			}
			c.eventCheck(data)
		case <-c.quit.Done(): //<-c.quit:
			return
		case <-c.timer.C:
			data := pong{c.token}
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
		var recieved ping
		err = json.Unmarshal(message, &recieved)
		if err != nil {
			log.WriteLog(logger.WARN, "an error occured during unmarshalling with websocket: "+c.ws.LocalAddr().String()+": "+err.Error())
			continue
		}
		//log.WriteLog(logger.INFO, fmt.Sprintf("ping token %s actual token %s", recieved.Token, c.token))
		if recieved.Token != c.token {
			log.WriteLog(logger.INFO, "Disconnecting websocket as it is a invalid token")
			//c.quit <- true
			c.quitFunc()
			return
		}
		c.ws.SetReadDeadline(time.Now().Add(pingDelay)) //screw handlers
		c.timer.Reset(pongDelay)                        //just in case if client pings in multiple intervals

	}
}

func (c *client) eventCheck(data interface{}) {
	switch data.(type) {
	case msg: //implement files soon or something idk guild change ban or kick whatever
		err := c.ws.WriteJSON(data)
		if err != nil {
			log.WriteLog(logger.ERROR, err.Error())
			//c.quit <- true
			c.quitFunc()
		}
	}
}

func webSocket(w http.ResponseWriter, r *http.Request) {

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

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	//defer ws.Close()
	if user.Id == 0 {
		reportError(http.StatusBadRequest, w, fmt.Errorf("invalid token: %v", token[0]))
		return
	}
	rows, err := db.Query("SELECT guild_id FROM userguilds WHERE user_id=$1", user.Id)
	if err != nil {
		reportError(http.StatusInternalServerError, w, err)
		return
	}
	var guilds []int
	for rows.Next() {
		var guild int
		rows.Scan(&guild)
		guilds = append(guilds, guild)
	}
	rows.Close()
	clients[user.Id] = make(brcastEvents)
	ctx := context.Background()
	quit, quitFunc := context.WithCancel(ctx)
	instanceuser := client{
		token:    token[0],
		ws:       ws,
		id:       user.Id,
		guilds:   guilds,
		timer:    time.NewTicker(pongDelay),
		quit:     quit,
		quitFunc: quitFunc,
	}
	instanceuser.run()

}
