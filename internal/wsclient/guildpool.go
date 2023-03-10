package wsclient

import (
	"context"
	"time"

	"github.com/asianchinaboi/backendserver/internal/config"
	"github.com/asianchinaboi/backendserver/internal/errors"
	"github.com/asianchinaboi/backendserver/internal/logger"
)

type addClientData struct {
	UniqueId string
	Ch       brcastEvents
}

type (
	removeClient chan string
	addClient    chan addClientData
)

type guildPool struct {
	guildId       int64
	clients       map[string]brcastEvents //channel of clients
	Remove        removeClient            //channel to broadcast which client to remove
	Add           addClient               //channel to broadcast which client to add
	Broadcast     brcastEvents            //broadcast to all clients
	Disconnecting bool
	quitCtx       context.Context
	quit          context.CancelFunc
	deadline      *time.Ticker
}

//bug
//if websocket is not connected
// and server is connected
//this pool still stays here

//solution
//create a timer to check if pool is empty - done

func (p *guildPool) run() {
	//this defer func sucks
	defer func() {
		p.Disconnecting = true //prevent any further signals since that causes more deadlocks
		p.quit()               //good practice
		p.deadline.Stop()      //stop timer

		//This is the only thing that prevents a channel deadlock
	OutterLoop: /*TODO: Find alternatives for this code may be unreliable (new answer: no) */
		for { //DO NOT TOUCH NEVER TOUCH
			//REMOVING THIS WILL BREAK ALL WEBSOCKET CODE WHEN LEAVING
			//THIS PREVENTS A DEADLOCK
			select {
			//purpose is to keep clearing the channels until it is empty
			case <-p.Remove:
			case <-p.Add:
			case <-p.Broadcast: //Dec 7 2022 editing code and still havent touched this til this day
			default: //legacy code basically
				break OutterLoop
			}
		} //welp it actually works christ
		//Dont remove this code or else everything will break
		Pools.RemoveFromGuildPool(p.guildId)
		//I think this is the only thing thats needed to prevent a deadlock since lockpool will release the lock
		close(p.Broadcast)
		close(p.Remove)
		close(p.Add)
		logger.Debug.Println("Stopped guild pool")
	}() //gracefully remove the pool when done
	for {
		select { //pretty sure a data race is impossible here
		case uid := <-p.Remove:
			delete(p.clients, uid) //uid is unique id
		case data := <-p.Add:
			p.clients[data.UniqueId] = data.Ch
		case data := <-p.Broadcast:
			Pools.BroadcastClientUIDMap(p.clients, data) // (BIG BAD BUG) problem this gets called before pool removal thus call on closed channel occurs
		case <-p.quitCtx.Done():
			return
		case <-p.deadline.C: //check if pool is empty every interval
			if len(p.clients) == 0 { //quit if no clients left in pool
				return //this leaves too early when there are values in channels still existing
				//but then lock isnt releasing which causes a deadlock
			}
		}
	}
}

func (p *pools) BroadcastGuild(guild int64, data DataFrame) error {
	p.guildsMutex.RLock() //prevents datarace
	defer p.guildsMutex.RUnlock()
	guildPool, ok := p.guilds[guild]
	if !ok {
		return errors.ErrGuildPoolNotExist
	}
	if guildPool.Disconnecting {
		return nil
	}
	guildPool.Broadcast <- data //stuck here
	return nil
}

func (p *pools) AddUserToGuildPool(guildId int64, userId int64) {
	p.guildsMutex.Lock()
	defer p.guildsMutex.Unlock()
	p.clientsMutex.RLock()
	defer p.clientsMutex.RUnlock()
	if _, ok := p.guilds[guildId]; !ok {
		quitCtx, quit := context.WithCancel(context.Background())
		deadline := time.NewTicker(config.Config.Guild.Timeout)
		pool := &guildPool{
			guildId:       guildId,
			clients:       make(map[string]brcastEvents),
			Remove:        make(removeClient),
			Add:           make(addClient),
			Broadcast:     make(brcastEvents),
			Disconnecting: false,
			quitCtx:       quitCtx,
			quit:          quit,
			deadline:      deadline,
		}
		p.guilds[guildId] = pool
		logger.Debug.Println("creating pools")
		go pool.run()
	}
	if p.guilds[guildId].Disconnecting {
		return
	}
	for uniqueId, broadcastChannel := range p.clients[userId] {
		clientData := addClientData{
			UniqueId: uniqueId,
			Ch:       broadcastChannel,
		}
		p.guilds[guildId].Add <- clientData
	}
}

func (p *pools) AddUIDToGuildPool(guildId int64, uid string, broadcast brcastEvents) {
	p.guildsMutex.Lock()
	defer p.guildsMutex.Unlock()
	p.clientsMutex.RLock()
	defer p.clientsMutex.RUnlock()
	pool, ok := p.guilds[guildId]
	clientData := addClientData{
		UniqueId: uid,
		Ch:       broadcast,
	}
	if !ok {
		quitCtx, quit := context.WithCancel(context.Background())
		logger.Debug.Printf("bruh %v \n", config.Config.Guild.Timeout)
		logger.Debug.Printf("wtf %v \n", config.Config)
		deadline := time.NewTicker(config.Config.Guild.Timeout)
		newPool := &guildPool{
			guildId:       guildId,
			clients:       make(map[string]brcastEvents),
			Remove:        make(removeClient),
			Add:           make(addClient),
			Broadcast:     make(brcastEvents),
			Disconnecting: false,
			quitCtx:       quitCtx,
			quit:          quit,
			deadline:      deadline,
		}
		p.guilds[guildId] = newPool
		go newPool.run()
		p.guilds[guildId].Add <- clientData
	} else {
		pool.Add <- clientData
	}
}

func (p *pools) RemoveUIDFromGuildPool(guildId int64, uid string) {
	p.guildsMutex.RLock()
	defer p.guildsMutex.RUnlock()
	p.clientsMutex.RLock()
	defer p.clientsMutex.RUnlock()
	_, ok := p.guilds[guildId]
	if !ok { // shouldnt happen but just in case
		logger.Warn.Println("tried to remove uid ", uid, " from guild", guildId, " however not found")
		return
	}
	p.guilds[guildId].Remove <- uid //RACE CONDITION This may be the cause of the websockets being stopped
}

func (p *pools) RemoveUserFromGuildPool(guildId int64, userId int64) { //most likely occurs here (WEBSOCKET BUG)
	p.guildsMutex.RLock()
	defer p.guildsMutex.RUnlock()
	p.clientsMutex.Lock()
	defer p.clientsMutex.Unlock()
	if _, ok := p.guilds[guildId]; !ok {
		logger.Warn.Printf("Tried to remove user:%d in guild: %d but guild pool does not exist", userId, guildId)
		return
	}
	if p.guilds[guildId].Disconnecting {
		logger.Debug.Println("guild disconnecting already")
		return
	}
	for uniqueId := range p.clients[userId] { // basically for uniqueId, _ := range clientAlias[user.Id]
		logger.Info.Printf("Removed user from pool %v", uniqueId)
		p.guilds[guildId].Remove <- uniqueId
	} //removes all instances of the client alias from the pool to avoid exploits
}

func (p *pools) RemoveFromGuildPool(guildId int64) {
	p.guildsMutex.Lock()
	defer p.guildsMutex.Unlock()
	delete(p.guilds, guildId)
}

func (p *pools) GetLengthGuilds() int {
	p.guildsMutex.RLock()
	defer p.guildsMutex.RUnlock()
	return len(p.guilds)
}
