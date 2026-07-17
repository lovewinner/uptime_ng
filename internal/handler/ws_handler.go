package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type Client struct {
	UserID uint
	Conn   *websocket.Conn
	Send   chan []byte
}

type WSHub struct {
	Clients    map[*Client]bool
	Broadcast  chan WSMessage
	Register   chan *Client
	Unregister chan *Client
	mu         sync.RWMutex
}

type WSMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
	UserID  uint        `json:"-"` // internal routing
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var Hub *WSHub

func NewWSHub() *WSHub {
	hub := &WSHub{
		Clients:    make(map[*Client]bool),
		Broadcast:  make(chan WSMessage, 256),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
	}
	Hub = hub
	return hub
}

func (h *WSHub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			h.Clients[client] = true
			h.mu.Unlock()

		case client := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.Clients[client]; ok {
				delete(h.Clients, client)
				close(client.Send)
			}
			h.mu.Unlock()

		case msg := <-h.Broadcast:
			h.mu.RLock()
			for client := range h.Clients {
				if msg.UserID == 0 || msg.UserID == client.UserID {
					select {
					case client.Send <- formatWSMessage(msg):
					default:
						close(client.Send)
						delete(h.Clients, client)
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *WSHub) SendToUser(userID uint, msgType string, payload interface{}) {
	h.Broadcast <- WSMessage{
		Type:    msgType,
		Payload: payload,
		UserID:  userID,
	}
}

func formatWSMessage(msg WSMessage) []byte {
	raw := map[string]interface{}{
		"type":    msg.Type,
		"payload": msg.Payload,
	}
	data, _ := json.Marshal(raw)
	return data
}

func (h *WSHub) HandleWebSocket(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("ws upgrade error: %v", err)
		return
	}

	client := &Client{
		UserID: userID.(uint),
		Conn:   conn,
		Send:   make(chan []byte, 256),
	}

	h.Register <- client

	go client.writePump()
	go client.readPump(h)
}

func (c *Client) writePump() {
	defer c.Conn.Close()
	for msg := range c.Send {
		if err := c.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return
		}
	}
}

func (c *Client) readPump(hub *WSHub) {
	defer func() {
		hub.Unregister <- c
		c.Conn.Close()
	}()

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}

		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		log.Printf("ws received from user %d: %v", c.UserID, msg)
		// Client-to-server messages will be handled in P3
	}
}