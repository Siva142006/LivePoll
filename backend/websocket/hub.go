package websocket

import (
	"log"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*Client]bool
}

type Client struct {
	Hub     *Hub
	Conn    *websocket.Conn
	PollID  string
	Send    chan []byte
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func NewHub() *Hub {
	return &Hub{clients: make(map[string]map[*Client]bool)}
}

func (h *Hub) Run() {}

func (h *Hub) Broadcast(pollID string, payload string) {
	h.mu.RLock()
	clients := h.clients[pollID]
	h.mu.RUnlock()
	for client := range clients {
		select {
		case client.Send <- []byte(payload):
		default:
			close(client.Send)
			h.mu.Lock()
			delete(clients, client)
			h.mu.Unlock()
		}
	}
}

func HandleWebSocket(c *gin.Context, hub *Hub, pollID string) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("websocket upgrade failed: %v", err)
		return
	}
	client := &Client{Hub: hub, Conn: conn, PollID: pollID, Send: make(chan []byte, 256)}
	hub.mu.Lock()
	if hub.clients[pollID] == nil {
		hub.clients[pollID] = make(map[*Client]bool)
	}
	hub.clients[pollID][client] = true
	hub.mu.Unlock()
	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.Conn.Close()
		c.Hub.mu.Lock()
		delete(c.Hub.clients[c.PollID], c)
		c.Hub.mu.Unlock()
	}()
	for {
		_, _, err := c.Conn.ReadMessage()
		if err != nil {
			return
		}
	}
}

func (c *Client) writePump() {
	defer c.Conn.Close()
	for msg := range c.Send {
		if err := c.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return
		}
	}
}
