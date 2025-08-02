package websocket

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/cortex-x/go-thai-id-card-reader/internal/domain"
	"github.com/gorilla/websocket"
)

type Client struct {
	conn   *websocket.Conn
	send   chan []byte
	hub    *Hub
	closed bool
	mu     sync.Mutex
}

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("Client registered. Total clients: %d", len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				h.mu.Unlock()
				log.Printf("Client unregistered. Total clients: %d", len(h.clients))
			} else {
				h.mu.Unlock()
			}

		case message := <-h.broadcast:
			h.mu.RLock()
			clients := make([]*Client, 0, len(h.clients))
			for client := range h.clients {
				clients = append(clients, client)
			}
			h.mu.RUnlock()

			for _, client := range clients {
				select {
				case client.send <- message:
				default:
					// Client's send channel is full, close it
					h.unregisterClient(client)
				}
			}
		}
	}
}

func (h *Hub) BroadcastMessage(messageType string, payload interface{}) error {
	msg := domain.WebSocketMessage{
		Type:    messageType,
		Payload: payload,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	h.broadcast <- data
	return nil
}

func (h *Hub) RegisterClient(conn *websocket.Conn) *Client {
	client := &Client{
		conn: conn,
		send: make(chan []byte, 256),
		hub:  h,
	}
	h.register <- client
	return client
}

func (h *Hub) unregisterClient(client *Client) {
	client.mu.Lock()
	if !client.closed {
		client.closed = true
		client.mu.Unlock()
		h.unregister <- client
	} else {
		client.mu.Unlock()
	}
}

func (c *Client) WritePump() {
	defer func() {
		_ = c.conn.Close()
	}()

	for message := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
			log.Printf("Error writing message: %v", err)
			return
		}
	}
	
	// The channel was closed, send close message
	_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
}

func (c *Client) ReadPump() {
	defer func() {
		c.hub.unregisterClient(c)
		_ = c.conn.Close()
	}()

	// We don't expect any messages from the client for this application
	// But we need to read to handle pings and connection close
	c.conn.SetReadLimit(512)
	
	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}
	}
}