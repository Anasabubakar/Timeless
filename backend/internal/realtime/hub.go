package realtime

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type EventType string

const (
	EventSponsorUpdated  EventType = "sponsor.updated"
	EventSponsorCreated  EventType = "sponsor.created"
	EventCampaignUpdated EventType = "campaign.updated"
	EventAgentCompleted  EventType = "agent.completed"
	EventNotification    EventType = "notification"
	EventPipelineMove    EventType = "pipeline.move"
	// EventActivity fires for every mutating request any member makes
	// (see middleware.AuditLog) — the org-wide "so-and-so just did X"
	// signal the realtime notification banner subscribes to. Deliberately
	// generic (one event type covering every resource) rather than one
	// event per entity kind: the banner only ever needs actor + verb +
	// entity type to render "jane@co.com updated Sponsors", and a single
	// type means the frontend doesn't need to know about new resource
	// kinds as they're added.
	EventActivity EventType = "activity"
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

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
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

// WebSocketHandler returns a Fiber handler using gorilla/websocket via adaptor
func (h *Hub) WebSocketHandler() fiber.Handler {
	httpHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		orgID := r.Context().Value("org_id")
		if orgID == nil {
			http.Error(w, "unauthorized", 401)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}

		client := &Client{
			ID:    uuid.New().String(),
			OrgID: orgID.(string),
			Conn:  conn,
			Send:  make(chan []byte, 256),
		}

		h.register <- client
		defer func() { h.unregister <- client }()

		go func() {
			for msg := range client.Send {
				if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
					return
				}
			}
		}()

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	})

	return adaptor.HTTPHandler(httpHandler)
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
