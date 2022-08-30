package main

import (
	"fmt"
	"net/http"

	"github.com/asianchinaboi/backendserver/logger"
)

type addClientData struct {
	UniqueId string
	Ch       brcastEvents
}

type poolBrcastEvents chan interface{}
type removeClient chan string
type addClient chan addClientData

type pool struct {
	guild         int
	clients       map[string]brcastEvents //channel of clients
	Remove        removeClient            //channel to broadcast which client to remove
	Add           addClient               //channel to broadcast which client to add
	Broadcast     poolBrcastEvents        //broadcast to all clients
	Disconnecting bool
}

func (p *pool) run() {
	defer func() {
		/*
			for len(p.Remove) > 0 {
				<-p.Remove
			}
			for len(p.Add) > 0 {
				<-p.Add
			}
			for len(p.Broadcast) > 0 {
				<-p.Broadcast
			}*/
		p.Disconnecting = true //prevent any further signals since that causes more deadlocks
	OutterLoop: /*TODO: Find alternatives for this code may be unreliable*/
		for { //DO NOT TOUCH NEVER TOUCH
			//REMOVING THIS WILL BREAK ALL WEBSOCKET CODE WHEN LEAVING
			//THIS PREVENTS A DEADLOCK
			select { //this is a test
			//purpose is to keep clearing the channels until it is empty
			case <-p.Remove:
			case <-p.Add:
			case <-p.Broadcast:
			default:
				break OutterLoop
			}
		} //welp it actually works christ

		//I think this is the only thing thats needed to prevent a deadlock since lockpool will release the lock
		lockPool.Lock()        //THIS IS THE CAUSE
		delete(pools, p.guild) //race condition here need FIX ASAP
		close(p.Broadcast)     //with pool:30 and client:86
		close(p.Remove)        //Knew it there is a panic that occurs sometimes when last user disconnects guild
		close(p.Add)           // this will bloody cause some nil pointer issue since i already closed da channel
		lockPool.Unlock()      // i think this fixes it
	}() //gracefully remove the pool when done
	for {
		select { //pretty sure a data race is impossible here
		case id := <-p.Remove:
			delete(p.clients, id)    //id is unique id
			if len(p.clients) == 0 { //quit if no clients left in pool
				return //this leaves too early when there are values in channels still existing
				//but then lock isnt releasing which causes a deadlock
			}
		case data := <-p.Add:
			p.clients[data.UniqueId] = data.Ch
		case data := <-p.Broadcast:
			lockAlias.Lock()
			for _, ch := range p.clients {
				ch <- data //closed channel fatal error FIX NOW
			}
			lockAlias.Unlock()
		}
	}
}

func broadcastGuild(guild int, data interface{}) (statusCode int, err error) {
	lockPool.Lock() //prevents datarace
	defer lockPool.Unlock()
	guildPool, ok := pools[guild]
	if !ok {
		return http.StatusBadRequest, errorGuildPoolNotExist
	}
	if guildPool.Disconnecting {
		return
	}
	guildPool.Broadcast <- data //stuck here
	return 0, nil
}

func createPool(guild int) {
	p := &pool{
		guild:         guild,
		clients:       make(map[string]brcastEvents),
		Remove:        make(removeClient),
		Add:           make(addClient),
		Broadcast:     make(poolBrcastEvents),
		Disconnecting: false,
	}
	pools[guild] = p
	go p.run()
}

func addUserToPool(guildId int, userId int) {
	lockPool.Lock()
	defer lockPool.Unlock()
	lockAlias.Lock()
	defer lockAlias.Unlock()
	if _, ok := pools[guildId]; !ok {
		log.WriteLog(logger.INFO, fmt.Sprintf("Canceled adding user %d to %d", userId, guildId))
		return
	}
	if pools[guildId].Disconnecting {
		return
	}
	for uniqueId, broadcastChannel := range clientAlias[userId] {
		clientData := addClientData{
			UniqueId: uniqueId,
			Ch:       broadcastChannel,
		}
		pools[guildId].Add <- clientData
	}
}

func removeUserFromPool(guildId int, userId int) { //most likely occurs here (WEBSOCKET BUG)
	lockPool.Lock()
	defer lockPool.Unlock()
	lockAlias.Lock()
	defer lockAlias.Unlock()
	if _, ok := pools[guildId]; !ok {
		log.WriteLog(logger.INFO, fmt.Sprintf("Canceled removing user %d to %d", userId, guildId))
		return
	}
	if pools[guildId].Disconnecting {
		return
	}
	for uniqueId := range clientAlias[userId] { // basically for uniqueId, _ := range clientAlias[user.Id]
		log.WriteLog(logger.INFO, "Removed user from pool "+uniqueId)
		pools[guildId].Remove <- uniqueId
	} //removes all instances of the client alias from the pool to avoid exploits
}
