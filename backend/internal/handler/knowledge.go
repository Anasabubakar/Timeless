package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/timeless/backend/internal/ai/memory"
	"github.com/timeless/backend/internal/middleware"
)

type KnowledgeHandler struct {
	store *memory.Store
}

func NewKnowledgeHandler(store *memory.Store) *KnowledgeHandler {
	return &KnowledgeHandler{store: store}
}

func (h *KnowledgeHandler) SemanticSearch(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	query := c.Query("q")
	if query == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Query parameter 'q' is required")
	}

	limitStr := c.Query("limit", "10")
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	nodeType := c.Query("type")

	nodes, err := h.store.SearchNodes(c.Context(), orgID, query, memory.NodeType(nodeType), limit)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Search failed: "+err.Error())
	}

	return c.JSON(fiber.Map{
		"data":  nodes,
		"count": len(nodes),
		"query": query,
	})
}

func (h *KnowledgeHandler) SearchMemories(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	query := c.Query("q")
	if query == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Query parameter 'q' is required")
	}

	limitStr := c.Query("limit", "10")
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	memories, err := h.store.SearchMemories(c.Context(), orgID, query, limit)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Memory search failed: "+err.Error())
	}

	return c.JSON(fiber.Map{
		"data":  memories,
		"count": len(memories),
		"query": query,
	})
}

type AddNodeRequest struct {
	NodeType   string                 `json:"node_type" validate:"required"`
	Name       string                 `json:"name" validate:"required"`
	EntityID   *string                `json:"entity_id"`
	Properties map[string]interface{} `json:"properties"`
}

func (h *KnowledgeHandler) AddNode(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)

	var req AddNodeRequest
	if err := c.Bind().JSON(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	if req.Name == "" || req.NodeType == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name and node_type are required")
	}

	node := &memory.KnowledgeNode{
		OrganizationID: orgID,
		NodeType:       memory.NodeType(req.NodeType),
		Name:           req.Name,
		Properties:     memory.JSON(req.Properties),
	}

	if req.EntityID != nil {
		id, err := uuid.Parse(*req.EntityID)
		if err == nil {
			node.EntityID = &id
		}
	}

	if err := h.store.AddNode(c.Context(), node); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to add node: "+err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(node)
}

type AddEdgeRequest struct {
	SourceID   string                 `json:"source_id" validate:"required"`
	TargetID   string                 `json:"target_id" validate:"required"`
	EdgeType   string                 `json:"edge_type" validate:"required"`
	Weight     float64                `json:"weight"`
	Properties map[string]interface{} `json:"properties"`
}

func (h *KnowledgeHandler) AddEdge(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)

	var req AddEdgeRequest
	if err := c.Bind().JSON(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	sourceID, err := uuid.Parse(req.SourceID)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid source_id")
	}

	targetID, err := uuid.Parse(req.TargetID)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid target_id")
	}

	weight := req.Weight
	if weight == 0 {
		weight = 1.0
	}

	edge := &memory.KnowledgeEdge{
		OrganizationID: orgID,
		SourceID:       sourceID,
		TargetID:       targetID,
		EdgeType:       memory.EdgeType(req.EdgeType),
		Weight:         weight,
		Properties:     memory.JSON(req.Properties),
	}

	if err := h.store.AddEdge(c.Context(), edge); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to add edge: "+err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(edge)
}

func (h *KnowledgeHandler) GetNeighbors(c fiber.Ctx) error {
	nodeID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid node ID")
	}

	depthStr := c.Query("depth", "1")
	depth, _ := strconv.Atoi(depthStr)
	if depth <= 0 || depth > 3 {
		depth = 1
	}

	nodes, edges, err := h.store.GetNeighbors(c.Context(), nodeID, depth)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to get neighbors: "+err.Error())
	}

	return c.JSON(fiber.Map{
		"nodes": nodes,
		"edges": edges,
	})
}

type StoreMemoryRequest struct {
	AgentType string                 `json:"agent_type" validate:"required"`
	Content   string                 `json:"content" validate:"required"`
	Metadata  map[string]interface{} `json:"metadata"`
}

func (h *KnowledgeHandler) StoreMemory(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)

	var req StoreMemoryRequest
	if err := c.Bind().JSON(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	if req.Content == "" || req.AgentType == "" {
		return fiber.NewError(fiber.StatusBadRequest, "content and agent_type are required")
	}

	entry, err := h.store.StoreMemory(c.Context(), orgID, req.AgentType, req.Content, req.Metadata)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to store memory: "+err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(entry)
}
