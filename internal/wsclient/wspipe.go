package wsclient

import (
	"encoding/json"
	"time"

	"github.com/asianchinaboi/backendserver/internal/config"
	"github.com/asianchinaboi/backendserver/internal/db"
	"github.com/asianchinaboi/backendserver/internal/errors"
	"github.com/asianchinaboi/backendserver/internal/events"
	"github.com/asianchinaboi/backendserver/internal/logger"
	"github.com/asianchinaboi/backendserver/internal/session"
)

func (c *wsClient) readPipe() {
	c.ws.SetReadDeadline(time.Now().Add(pingDelay)) //note to self put that thing in seconds otherwise its goddamn miliseconds which is hard to debug
	for {                                           //need to check for quit
		_, message, err := c.ws.ReadMessage()
		if err != nil { //should usually return io error which is fine since it means the websocket has timeouted
			logger.Error.Println(err) //or if the websocket has closed which is a 1000 (normal)
			//c.quit <- true
			c.quit() //if recieve websocket closed error then you gotta do what you gotta do
			logger.Info.Println("Disconnecting websocket")
			return
		}
		//		log.WriteLog(logger.INFO, "string json:"+string(message))
		var received DataFrame
		err = json.Unmarshal(message, &received)
		if err != nil {
			logger.Warn.Printf("an error occured during unmarshalling with websocket: %v: %v", c.ws.LocalAddr().String(), err.Error())
			continue
		}
		c.readData(received)
	}
}

func (c *wsClient) writePipe() {
	for {
		select {
		//recieve messages
		case data, ok := <-c.broadcast:
			//in the future check if c.broadcast is ever closed if not remove lines 42-46
			if !ok { //idk if this is needed or not
				c.quit() //call cancel but never actually recieve it
				return   //the channel has been closed get out of here
			}
			c.ws.WriteJSON(data)

			if data.Event == events.LOG_OUT { //maybe find other solutions later
				c.quit()
				return
			}
		case <-c.quitctx.Done(): //<-c.quit:
			return
		}
	}
}

func (c *wsClient) readData(body DataFrame) {
	//logger.Debug.Println("worked?") //most fucking annoying line ever
	switch body.Op {
	case TYPE_HEARTBEAT:
		c.ws.SetReadDeadline(time.Now().Add(pingDelay))
		res := DataFrame{
			Op: TYPE_HEARTBEATACK,
		}
		c.ws.WriteJSON(res)
	case TYPE_IDENTIFY:
		if c.id > 0 {
			c.quit()
			return
		}
		bytes, err := json.Marshal(body.Data) //shit but temporary for now until I find another solution
		if err != nil {                       //to convert inner into interface and not map[string]interface{}
			logger.Error.Println(err)
			c.quit()
			return
		}
		var data helloResFrame
		err = json.Unmarshal(bytes, &data)
		if err != nil {
			logger.Error.Println(err)
			c.quit()
			return
		}
		Token := data.Token
		user, err := session.CheckToken(Token)
		if err != nil {
			logger.Error.Println(err)
			c.quit()
			return
		}

		c.deadlineCancel()
		c.id = user.Id
		if Pools.GetLengthForClient(c.id) >= config.Config.User.WSPerUser {
			logger.Error.Println(errors.ErrSessionTooManySessions)
			c.quit()
			return
		}
		go c.tokenExpireDeadline(user.Expires)
		c.uniqueId = session.GenerateRandString(32)

		rows, err := db.Db.Query("SELECT guild_id FROM userguilds WHERE user_id=$1", c.id)
		if err != nil {
			logger.Error.Println(err)
			c.quit()
			return
		}
		//var guilds []int

		for rows.Next() {
			var guild int64
			rows.Scan(&guild)
			//	guilds = append(guilds, guild)
			Pools.AddUIDToGuildPool(guild, c.uniqueId, c.broadcast)
		}

		//		c.guilds = guilds
		//get all the guilds the user is in
		rows.Close()
		Pools.AddUserToClientPool(c.id, c.uniqueId, c.broadcast)
		logger.Info.Println("added to client pool")
		res := DataFrame{
			Op: TYPE_READY,
		}
		c.ws.WriteJSON(res)
	default:
		logger.Warn.Printf("Invalid Op: %v\n", body.Op)
	}
}
