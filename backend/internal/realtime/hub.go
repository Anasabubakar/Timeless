package realtime

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gofiber/websocket"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type EventType string

const (
	EventSponsorUpdated  EventType = "sponsor.updated"
	EventSponsorCreated  EventType = "sponsor.created"
	EventCampaignUpdated EventType = "campaign.updated"
	EventAgentCompleted  EventType = "agent.completed"
	EventNotification    EventType = "notification"
	EventPipelineMove    EventType = "pipeline.move"
)

type Event struct {
	Type      EventType              `json:"type"`
	Payload   map[string]interface{} `json:"payload"`
	OrgID     string                 `json:"org_id"`
	Timestamp time.Time              `json:"timestamp"`
}

type Client struct {
	ID    string
	OrgID string
	Conn  *websocket.Conn
	Send  chan []byte
}

type Hub struct {
	mu         sync.RWMutex
	clients    map[string]*Client
	orgClients map[string]map[string]*Client
	broadcast  chan *Event
	register   chan *Client
	unregister chan *Client
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		orgClients: make(map[string]map[string]*Client),
		broadcast:  make(chan *Event, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.ID] = client
			if _, ok := h.orgClients[client.OrgID]; !ok {
				h.orgClients[client.OrgID] = make(map[string]*Client)
			}
			h.orgClients[client.OrgID][client.ID] = client
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.ID]; ok {
				delete(h.clients, client.ID)
				if orgMap, ok := h.orgClients[client.OrgID]; ok {
					delete(orgMap, client.ID)
					if len(orgMap) == 0 {
						delete(h.orgClients, client.OrgID)
					}
				}
				close(client.Send)
			}
			h.mu.Unlock()

		case event := <-h.broadcast:
			data, _ := json.Marshal(event)
			h.mu.RLock()
			if clients, ok := h.orgClients[event.OrgID]; ok {
				for _, client := range clients {
					select {
					case client.Send <- data:
					default:
						go func(c *Client) { h.unregister <- c }(client)
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) Publish(event *Event) {
	event.Timestamp = time.Now()
	h.broadcast <- event
}

func (h *Hub) WebSocketHandler() fiber.Handler {
	return websocket.New(func(c *websocket.Conn) {
		orgID := c.Locals("org_id")
		if orgID == nil {
			c.Close()
			return
		}

		client := &Client{
			ID:    uuid.New().String(),
			OrgID: orgID.(string),
			Conn:  c,
			Send:  make(chan []byte, 256),
		}

		h.register <- client
		defer func() { h.unregister <- client }()

		go func() {
			for msg := range client.Send {
				if err := c.WriteMessage(websocket.TextMessage, msg); err != nil {
					return
				}
			}
		}()

		for {
			_, _, err := c.ReadMessage()
			if err != nil {
				break
			}
		}
	})
}

func (h *Hub) SSEHandler() fiber.Handler {
	return func(c fiber.Ctx) error {
		orgID := c.Locals("org_id")
		if orgID == nil {
			return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
		}

		c.Set("Content-Type", "text/event-stream")
		c.Set("Cache-Control", "no-cache")
		c.Set("Connection", "keep-alive")

		client := &Client{
			ID:    uuid.New().String(),
			OrgID: orgID.(string),
			Send:  make(chan []byte, 256),
		}

		h.register <- client
		defer func() { h.unregister <- client }()

		ctx := c.Context()
		for {
			select {
			case msg, ok := <-client.Send:
				if !ok {
					return nil
				}
				if _, err := c.Write([]byte("data: " + string(msg) + "\n\n")); err != nil {
					return nil
				}
			case <-ctx.Done():
				return nil
			}
		}
	}
}
