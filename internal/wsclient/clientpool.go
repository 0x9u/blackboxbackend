package wsclient

import (
	"github.com/asianchinaboi/backendserver/internal/errors"
	"github.com/asianchinaboi/backendserver/internal/events"
	"github.com/asianchinaboi/backendserver/internal/logger"
)

func (p *pools) AddUserToClientPool(id int, uid string, broadcast brcastEvents) {
	p.clientsMutex.Lock() //prevents datarace
	defer p.clientsMutex.Unlock()
	if _, ok := p.clients[id]; !ok {
		p.clients[id] = make(map[string]brcastEvents)
	}

	p.clients[id][uid] = broadcast
}

func (p *pools) removeUserUIDFromClientPool(id int, uid string) {
	p.clientsMutex.Lock()
	defer p.clientsMutex.Unlock()
	if _, ok := p.clients[id]; !ok {
		logger.Warn.Println("Requested to remove ID: ", id, "UID: ", uid, " however was not found")
		return
	}
	delete(p.clients[id], uid)

	if len(p.clients[id]) == 0 {
		delete(p.clients, id)
	}
	//logger.Debug.Printf("apple pie %v \n", p.clients)
}

func (p *pools) DisconnectUserFromClientPool(id int) {
	p.clientsMutex.Lock()
	defer p.clientsMutex.Unlock()
	clientList, ok := p.clients[id] //problem if two websockets of same user exist only of those two will be sent
	if !ok {
		logger.Warn.Println("Requested to disconnect ID: ", id, " however was not found")
		return
	}
	for _, client := range clientList {
		client <- DataFrame{
			Op:    TYPE_DISPATCH,
			Event: events.LOG_OUT,
		}
	} //will automatically delete itself from defer funciton in wsclient through RemoveUserUIDFromClientPool
}

func (p *pools) BroadcastClient(id int, data DataFrame) error {
	p.clientsMutex.RLock()
	defer p.clientsMutex.RUnlock()
	clientList, ok := p.clients[id] //problem if two websockets of same user exist only of those two will be sent
	if !ok {
		logger.Debug.Println("bad shit")
		return errors.ErrUserClientNotExist
	}
	for _, client := range clientList {
		client <- data
	}
	return nil
}

func (p *pools) BroadcastClientUIDMap(clients map[string]brcastEvents, data DataFrame) {
	p.clientsMutex.RLock()
	defer p.clientsMutex.RUnlock()
	for _, ch := range clients {
		ch <- data //closed channel fatal error FIX NOW
	}
}

func (p *pools) GetLengthClients() int {
	p.clientsMutex.RLock()
	defer p.clientsMutex.RUnlock()
	return len(p.clients)
}
