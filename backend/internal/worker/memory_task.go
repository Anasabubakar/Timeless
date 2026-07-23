package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"

	"github.com/sponsoros/backend/internal/ai/memory"
	"github.com/sponsoros/backend/internal/ai/provider"
	"gorm.io/gorm"
)

type MemoryIndexHandler struct {
	logger *slog.Logger
	db     *gorm.DB
	store  *memory.Store
}

func NewMemoryIndexHandler(logger *slog.Logger, db *gorm.DB, embedder provider.Embedder) *MemoryIndexHandler {
	return &MemoryIndexHandler{
		logger: logger,
		db:     db,
		store:  memory.NewStore(db, embedder),
	}
}

type MemoryIndexPayload struct {
	OrgID      string `json:"org_id"`
	EntityType string `json:"entity_type"`
	EntityID   string `json:"entity_id"`
	Content    string `json:"content"`
}

func (h *MemoryIndexHandler) Handle(ctx context.Context, t *asynq.Task) error {
	var payload MemoryIndexPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal memory index payload: %w", err)
	}

	orgID := uuidFromString(payload.OrgID)

	if payload.Content == "" {
		h.logger.Warn("empty content for memory indexing, skipping",
			"entity_type", payload.EntityType,
			"entity_id", payload.EntityID,
		)
		return nil
	}

	metadata := map[string]interface{}{
		"entity_type": payload.EntityType,
		"entity_id":   payload.EntityID,
	}

	_, err := h.store.StoreMemory(ctx, orgID, "system", payload.Content, metadata)
	if err != nil {
		h.logger.Error("failed to store memory with embedding",
			"entity_type", payload.EntityType,
			"entity_id", payload.EntityID,
			"error", err,
		)
		return nil
	}

	node := &memory.KnowledgeNode{
		OrganizationID: orgID,
		NodeType:       memory.NodeType(payload.EntityType),
		Name:           payload.Content[:min(len(payload.Content), 255)],
	}
	entityUUID := uuidFromString(payload.EntityID)
	node.EntityID = &entityUUID

	if err := h.store.AddNode(ctx, node); err != nil {
		h.logger.Error("failed to add knowledge node",
			"entity_type", payload.EntityType,
			"entity_id", payload.EntityID,
			"error", err,
		)
	}

	h.logger.Info("memory indexed with embeddings",
		"org_id", payload.OrgID,
		"entity_type", payload.EntityType,
		"entity_id", payload.EntityID,
	)
	return nil
}

func RegisterMemoryIndexHandler(mux *asynq.ServeMux, logger *slog.Logger, db *gorm.DB, embedder provider.Embedder) {
	h := NewMemoryIndexHandler(logger, db, embedder)
	mux.HandleFunc("memory:index:v2", h.Handle)
}
