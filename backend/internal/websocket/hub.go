package websocket

// hub.go — WebSocket event hub.
// Frontend clients connect to /api/events/ws; the hub broadcasts events pushed
// from the OCPP backend or generated internally.

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/yourorg/csms-backend/internal/auth"
	"github.com/yourorg/csms-backend/internal/model"
)

var upgrader = websocket.Upgrader{
	CheckOrigin:     func(r *http.Request) bool { return true }, // tighten in production
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type client struct {
	conn *websocket.Conn
	send chan []byte
}

type Hub struct {
	mu      sync.RWMutex
	clients map[*client]struct{}
	eventCh chan *model.Event
}

func NewHub(eventCh chan *model.Event) *Hub {
	return &Hub{
		clients: make(map[*client]struct{}),
		eventCh: eventCh,
	}
}

// Run starts the broadcast loop — call in a goroutine
func (h *Hub) Run() {
	for ev := range h.eventCh {
		b, _ := json.Marshal(ev)
		h.mu.RLock()
		for c := range h.clients {
			select {
			case c.send <- b:
			default:
				// slow client — drop message
			}
		}
		h.mu.RUnlock()
	}
}

// ServeWS upgrades the HTTP connection and registers the client
func (h *Hub) ServeWS(c *gin.Context) {
	// Auth via query token (Bearer in WS isn't standard)
	token := c.Query("token")
	if token == "" {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	if _, err := auth.ParseToken(token); err != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[ws] upgrade: %v", err)
		return
	}

	cl := &client{conn: conn, send: make(chan []byte, 256)}
	h.mu.Lock()
	h.clients[cl] = struct{}{}
	h.mu.Unlock()

	// Write pump
	go func() {
		defer func() {
			conn.Close()
			h.mu.Lock()
			delete(h.clients, cl)
			h.mu.Unlock()
		}()
		for msg := range cl.send {
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		}
	}()

	// Read pump (discard, just keep alive)
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
	close(cl.send)
}
