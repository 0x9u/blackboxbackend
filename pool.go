package main

import (
	"fmt"
	"net/http"

	"github.com/asianchinaboi/backendserver/logger"
)

type addClientData struct {
	Id int
	Ch brcastEvents
}

type poolBrcastEvents chan interface{}
type removeClient chan int
type addClient chan addClientData

type pool struct {
	guild     int
	clients   map[int]brcastEvents //channel of clients
	Remove    removeClient         //channel to broadcast which client to remove
	Add       addClient            //channel to broadcast which client to add
	Broadcast poolBrcastEvents     //broadcast to all clients
}

func (p *pool) run() {
	defer func() {
		log.WriteLog(logger.INFO, fmt.Sprintf("Pool closed: %d", p.guild))
		delete(pools, p.guild) //race condition here need FIX ASAP
		close(p.Broadcast)     //with pool:30 and client:73
		close(p.Remove)
		close(p.Add)
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
			p.clients[data.Id] = data.Ch
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
		clients:   make(map[int]brcastEvents),
		Remove:    make(removeClient),
		Add:       make(addClient),
		Broadcast: make(poolBrcastEvents),
	}
	pools[guild] = p
	log.WriteLog(logger.INFO, fmt.Sprintf("Pool created for guild %d", guild))
	go p.run()
}
