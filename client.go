package main

import (
	"github.com/asianchinaboi/backendserver/logger"
	"github.com/gorilla/websocket"
)

type brcastEvents chan interface{}

type client struct {
	ws          *websocket.Conn
	id          int
	guilds      []int
	quit        chan bool
	broadcaster brcastEvents
}

func (c *client) run() {
	defer c.ws.Close()
	for {
		select {
		//case : //recieve messages
		case data := <-c.broadcaster:
			c.eventCheck(data)
		case <-c.quit:
			close(c.broadcaster)
			return
			//default: //get hearbeat and time for every 10 seconds
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
