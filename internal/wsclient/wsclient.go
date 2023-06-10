package wsclient

import (
	"context"
	"time"

	"github.com/asianchinaboi/backendserver/internal/db"
	"github.com/asianchinaboi/backendserver/internal/logger"
	"github.com/gorilla/websocket"
)

type brcastEvents chan DataFrame
type wsClient struct {
	ws       *websocket.Conn
	id       int64
	uniqueId string //since some guys might be using multiple connections on one account
	//	guilds         []int  //not used might remove later
	broadcast      brcastEvents
	quitctx        context.Context //chan bool //also temporary maybe use contexts later
	quit           context.CancelFunc
	deadline       context.Context
	deadlineCancel context.CancelFunc
	//	keepAlive bool //temporary try find solution
}

func (c *wsClient) Run() {
	defer func() {
		if err := c.ws.Close(); err != nil {
			logger.Warn.Println("an error occured when leaving websocket: ", err)
			return
		}

		logger.Info.Println("Websocket of " + c.ws.LocalAddr().String() + " has been closed")
		//leaves the guild pools
		rows, err := db.Db.Query("SELECT guild_id FROM userguilds WHERE user_id = $1 AND banned = false", c.id)
		if err != nil {
			logger.Error.Printf("an error occured when getting guilds of user %v\n", err)
			return
		}
		for rows.Next() { //should be using guilds array instead lol
			var guildId int64
			err = rows.Scan(&guildId)
			if err != nil {
				logger.Error.Printf("an error occured when getting guilds of user %v\n", err)
				return
			}

			Pools.RemoveUserFromGuildPool(guildId, c.id)
		}

		//moved line here to stop close channel errors
		Pools.removeUserUIDFromClientPool(c.id, c.uniqueId) // dont move this line above where guilds is deleted
		//order is extremely important

		close(c.broadcast) //close of nil channel error occurs here sometimes
		c.deadlineCancel()
		logger.Debug.Println("wsclient ended")
	}()
	logger.Info.Println("Websocket active of " + c.ws.LocalAddr().String())

	c.hello()
	go c.tokenDeadline()
	go c.readPipe()
	c.writePipe()
}

func (c *wsClient) tokenDeadline() {
	<-c.deadline.Done() //crash sometimes happens here (invalid memory address or nil pointer dereference)
	logger.Info.Println("deadline passed")
	if c.deadline.Err() != context.Canceled {
		logger.Info.Println("cancelling from deadline")
		c.quit()
		return
	}
	logger.Info.Println("deadline passed end")

}

func (c *wsClient) tokenExpireDeadline(expireTime int64) { //i have no idea if this works
	timeLeft := time.Until(time.Now().Add(time.Duration(expireTime) * time.Second))
	select {
	case <-time.After(timeLeft):
		c.quit()
	case <-c.quitctx.Done():
		return
	}

}

func (c *wsClient) hello() {
	body := DataFrame{
		Op: TYPE_HELLO,
		Data: helloFrame{
			HeartbeatInterval: int(pingDelay / time.Second * 1000),
		},
		Event: "",
	}
	err := c.ws.WriteJSON(body)
	if err != nil {
		logger.Error.Println(err)
		c.quit()
		return
	}
	deadline, cancelFunc := context.WithTimeout(context.Background(), tokenTimeout)
	c.deadline = deadline
	c.deadlineCancel = cancelFunc
}

func NewWsClient(ws *websocket.Conn) (*wsClient, error) {
	ctx := context.Background()
	quit, quitFunc := context.WithCancel(ctx)
	instanceuser := wsClient{
		ws:        ws,
		id:        0,  //user id will be received when user sends identify payload
		uniqueId:  "", //uniqueId,
		broadcast: make(brcastEvents),
		quitctx:   quit,
		quit:      quitFunc,
	}
	return &instanceuser, nil
}
