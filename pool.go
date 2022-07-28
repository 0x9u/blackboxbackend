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
	guild     int
	clients   map[string]brcastEvents //channel of clients
	Remove    removeClient            //channel to broadcast which client to remove
	Add       addClient               //channel to broadcast which client to add
	Broadcast poolBrcastEvents        //broadcast to all clients
}

func (p *pool) run() {
	defer func() {
		log.WriteLog(logger.INFO, fmt.Sprintf("Pool closed: %d", p.guild))
		delete(pools, p.guild) //race condition here need FIX ASAP
		close(p.Broadcast)     //with pool:30 and client:86
		close(p.Remove)        //Knew it there is a panic that occurs sometimes when last user disconnects guild
		close(p.Add)           // this will bloody cause some nil pointer issue since i already closed da channel
	}() //gracefully remove the pool when done
	for {
		select { //pretty sure a data race is impossible here
		case id := <-p.Remove:
			log.WriteLog(logger.INFO, "User left pool")
			delete(p.clients, id)
			if len(p.clients) == 0 { //quit if no clients left in pool
				return
			}
		case data := <-p.Add:
			p.clients[data.UniqueId] = data.Ch
		case data := <-p.Broadcast:
			for _, ch := range p.clients {
				ch <- data
			}
		}
	}
}

func broadcastGuild(guild int, data interface{}) (statusCode int, err error) {
	guildPool, ok := pools[guild]
	if !ok {
		return http.StatusBadRequest, errorGuildPoolNotExist
	}
	guildPool.Broadcast <- data
	return 0, nil
}

func createPool(guild int) {
	p := &pool{
		guild:     guild,
		clients:   make(map[string]brcastEvents),
		Remove:    make(removeClient),
		Add:       make(addClient),
		Broadcast: make(poolBrcastEvents),
	}
	pools[guild] = p
	log.WriteLog(logger.INFO, fmt.Sprintf("Pool created for guild %d", guild))
	go p.run()
}
