package realtime

import (
	"sync"
	"time"
)

type ClientHandle struct {
	User   string
	IP     string
	Scope  Scope
	SendCh chan<- Event
	OnDrop func(reason string)
}

type client struct {
	user   string
	ip     string
	scope  Scope
	sendCh chan<- Event
	onDrop func(reason string)
}

type hub struct {
	mu      sync.RWMutex
	clients map[*client]struct{}
	byUser  map[string]int
	byIP    map[string]int
}

var defaultHub = newHub()

func newHub() *hub {
	return &hub{
		clients: map[*client]struct{}{},
		byUser:  map[string]int{},
		byIP:    map[string]int{},
	}
}

func Register(c *ClientHandle) (unregister func()) {
	return defaultHub.Register(c)
}

func Publish(topic Topic, payload interface{}) {
	defaultHub.Publish(topic, payload)
}

func CloseAll(reason string) {
	defaultHub.CloseAll(reason)
}

func Reserve(user string, ip string, maxPerUser int, maxPerIP int) (release func(), ok bool) {
	return defaultHub.Reserve(user, ip, maxPerUser, maxPerIP)
}

func (h *hub) Reserve(user string, ip string, maxPerUser int, maxPerIP int) (release func(), ok bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if maxPerUser > 0 && h.byUser[user] >= maxPerUser {
		return func() {}, false
	}
	if maxPerIP > 0 && h.byIP[ip] >= maxPerIP {
		return func() {}, false
	}
	h.byUser[user]++
	h.byIP[ip]++

	var once sync.Once
	return func() {
		once.Do(func() {
			h.release(user, ip)
		})
	}, true
}

func (h *hub) Register(c *ClientHandle) (unregister func()) {
	if c == nil || c.SendCh == nil {
		return func() {}
	}
	internal := &client{
		user:   c.User,
		ip:     c.IP,
		scope:  c.Scope,
		sendCh: c.SendCh,
		onDrop: c.OnDrop,
	}
	h.mu.Lock()
	h.clients[internal] = struct{}{}
	h.mu.Unlock()

	var once sync.Once
	return func() {
		once.Do(func() {
			h.mu.Lock()
			delete(h.clients, internal)
			h.mu.Unlock()
		})
	}
}

func (h *hub) Publish(topic Topic, payload interface{}) {
	event := Event{
		Type:    topic,
		Ts:      time.Now().Unix(),
		Payload: payload,
	}
	clients := h.snapshot(topic)
	for _, c := range clients {
		select {
		case c.sendCh <- event:
		default:
			h.drop(c, "slow")
		}
	}
}

func (h *hub) CloseAll(reason string) {
	h.mu.Lock()
	clients := make([]*client, 0, len(h.clients))
	for c := range h.clients {
		clients = append(clients, c)
	}
	h.clients = map[*client]struct{}{}
	h.mu.Unlock()

	for _, c := range clients {
		c.callDrop(reason)
	}
}

func (h *hub) release(user string, ip string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.byUser[user] > 0 {
		h.byUser[user]--
	}
	if h.byIP[ip] > 0 {
		h.byIP[ip]--
	}
}

func (h *hub) snapshot(topic Topic) []*client {
	h.mu.RLock()
	defer h.mu.RUnlock()
	clients := make([]*client, 0, len(h.clients))
	for c := range h.clients {
		if topicAllowed(topic, c.scope) {
			clients = append(clients, c)
		}
	}
	return clients
}

func (h *hub) drop(c *client, reason string) {
	h.mu.Lock()
	_, ok := h.clients[c]
	if ok {
		delete(h.clients, c)
	}
	h.mu.Unlock()
	if ok {
		c.callDrop(reason)
	}
}

func (c *client) callDrop(reason string) {
	if c.onDrop != nil {
		c.onDrop(reason)
	}
}
