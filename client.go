package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/asianchinaboi/backendserver/logger"
	"github.com/gorilla/websocket"
)

type brcastEvents chan interface{}

type client struct {
	token  string
	ws     *websocket.Conn
	id     int
	guilds []int
	quit   chan bool
}

type ping struct {
	Token string `json:"token"`
}

const (
	heartbeatDelay = 10
	messageLimit   = 2 << 7
)

func (c *client) run() {
	defer c.ws.Close()
	go c.heartBeat()
	c.ws.SetReadLimit(messageLimit)
	for {
		select {
		//case : //recieve messages
		case data, ok := <-clients[c.id]:
			if !ok {
				err := c.ws.Close()
				if err != nil {
					log.WriteLog(logger.ERROR, fmt.Errorf("an error occured when leaving websocket %v", err).Error())
				}
				log.WriteLog(logger.INFO, "Websocket of "+c.ws.LocalAddr().String()+" has been closed")
				return
			}
			c.eventCheck(data)
		case <-c.quit:
			err := c.ws.Close()
			if err != nil {
				log.WriteLog(logger.ERROR, fmt.Errorf("an error occured when leaving websocket %v", err).Error())
			}
			close(clients[c.id])
			delete(clients, c.id)
			log.WriteLog(logger.INFO, "Leaving websocket")
			return
		}
	}
}

func (c *client) heartBeat() {
	c.ws.SetReadDeadline(time.Now().Add(heartbeatDelay * time.Second)) //note to self put that thing in seconds otherwise its goddamn miliseconds which is hard to debug
	/*c.ws.SetPingHandler(func(string) error {
		c.ws.SetReadDeadline(time.Now().Add(heartbeatDelay * time.Second))
		return nil
	}) //copied from example
	*/
	for {
		/*err := c.ws.WriteMessage(websocket.PingMessage, []byte("ping"))
		if err != nil {
			c.quit <- true
			return
		}
		*/
		_, message, err := c.ws.ReadMessage()
		if err != nil { //should usually return io error which is fine since it means the websocket has timeouted
			log.WriteLog(logger.ERROR, err.Error()) //or if the websocket has closed which is a 1000 (normal)
			c.quit <- true
			log.WriteLog(logger.INFO, "Disconnecting websocket")
			break
		}
		var recieved ping
		err = json.Unmarshal(message, &recieved)
		if err != nil {
			log.WriteLog(logger.WARN, "an error occured during unmarshalling with websocket: "+c.ws.LocalAddr().String()+": "+err.Error())
			continue
		}
		log.WriteLog(logger.INFO, fmt.Sprintf("ping token %s actual token %s", recieved.Token, c.token))
		if recieved.Token != c.token {
			log.WriteLog(logger.INFO, "Disconnecting websocket as it is a invalid token")
			c.quit <- true
			return
		}

	}
}

func (c *client) eventCheck(data interface{}) {
	switch data.(type) {
	case msg:
		var content msg
		err := c.ws.WriteJSON(content)
		if err != nil {
			log.WriteLog(logger.ERROR, err.Error())
			c.quit <- true
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
	instanceuser := client{
		token:  token[0],
		ws:     ws,
		id:     user.Id,
		guilds: guilds,
		quit:   make(chan bool),
	}
	instanceuser.run()

}
