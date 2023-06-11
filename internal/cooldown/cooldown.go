package cooldown

import (
	"sync"
	"time"

	"github.com/asianchinaboi/backendserver/internal/config"
	"github.com/asianchinaboi/backendserver/internal/logger"
)

//TODO: test this

//sorta works ig?
//idk

type Limiter struct {
	ip       string
	maxCount int //max amount of requests
	count    int
	ch       chan struct{}
	ticker   *time.Ticker
	deadline *time.Timer
}

var Manager *manager

type manager struct {
	sync.RWMutex
	limiters map[string]*Limiter
}

func (l *Limiter) run() { //6 am code
	defer func() {
		logger.Debug.Println("Stopped limiter")
	}()
	logger.Debug.Println("Started limiter")
	for {
		select {
		case <-l.ch:
			logger.Debug.Println("received a msg to add")
			l.count--
		case <-l.ticker.C:
			l.count = l.maxCount
		case <-l.deadline.C:
			return
		}
	}
}

func (m *manager) AddCount(ip string) bool { //idk if this works (returns true if can else false)
	m.RWMutex.Lock()
	defer m.RWMutex.Unlock()
	limiter, ok := m.limiters[ip]
	if !ok {
		limiter = &Limiter{
			ip:       ip,
			maxCount: 10,
			count:    10,
			ch:       make(chan struct{}),
			ticker:   time.NewTicker(config.Config.User.CoolDownLength),
			deadline: time.NewTimer(60 * time.Second),
		}
		m.limiters[ip] = limiter
		logger.Debug.Println("Created new limiter")
		go limiter.run()
	}
	limiter.deadline.Reset(60 * time.Second)
	if limiter.count <= 0 {
		return false
	}
	logger.Debug.Println("before send chan")
	limiter.ch <- struct{}{}
	logger.Debug.Println("after send chan")
	return true
}

func (m *manager) removeLimiter(ip string) {
	m.RWMutex.Lock()
	defer m.RWMutex.Unlock()
	delete(m.limiters, ip)
}

func init() {
	Manager = &manager{
		limiters: make(map[string]*Limiter),
	}
}
