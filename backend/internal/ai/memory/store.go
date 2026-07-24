package memory

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sponsoros/backend/internal/ai/provider"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type NodeType string

const (
	NodeCompany    NodeType = "company"
	NodeContact    NodeType = "contact"
	NodeCampaign   NodeType = "campaign"
	NodeSponsor    NodeType = "sponsor"
	NodeEvent      NodeType = "event"
	NodeInsight    NodeType = "insight"
	NodePreference NodeType = "preference"
)

type EdgeType string

const (
	EdgeWorksAt     EdgeType = "works_at"
	EdgeSponsors    EdgeType = "sponsors"
	EdgeRelatedTo   EdgeType = "related_to"
	EdgeMentionedIn EdgeType = "mentioned_in"
	EdgePrefers     EdgeType = "prefers"
	EdgeInfluences  EdgeType = "influences"
)

type KnowledgeNode struct {
	ID             uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OrganizationID uuid.UUID  `json:"organization_id" gorm:"type:uuid;not null;index"`
	NodeType       NodeType   `json:"node_type" gorm:"type:varchar(50);not null;index"`
	EntityID       *uuid.UUID `json:"entity_id,omitempty" gorm:"type:uuid;index"`
	Name           string     `json:"name" gorm:"type:varchar(255);not null"`
	Properties     JSON       `json:"properties" gorm:"type:jsonb;default:'{}'"`
	Embedding      Vector     `json:"embedding,omitempty" gorm:"type:vector(1536)"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type KnowledgeEdge struct {
	ID             uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OrganizationID uuid.UUID `json:"organization_id" gorm:"type:uuid;not null;index"`
	SourceID       uuid.UUID `json:"source_id" gorm:"type:uuid;not null;index"`
	TargetID       uuid.UUID `json:"target_id" gorm:"type:uuid;not null;index"`
	EdgeType       EdgeType  `json:"edge_type" gorm:"type:varchar(50);not null;index"`
	Weight         float64   `json:"weight" gorm:"default:1.0"`
	Properties     JSON      `json:"properties" gorm:"type:jsonb;default:'{}'"`
	CreatedAt      time.Time `json:"created_at"`
}

type MemoryEntry struct {
	ID             uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OrganizationID uuid.UUID `json:"organization_id" gorm:"type:uuid;not null;index"`
	AgentType      string    `json:"agent_type" gorm:"type:varchar(50);not null;index"`
	Content        string    `json:"content" gorm:"type:text;not null"`
	Summary        string    `json:"summary" gorm:"type:text"`
	Embedding      Vector    `json:"embedding,omitempty" gorm:"type:vector(1536)"`
	Metadata       JSON      `json:"metadata" gorm:"type:jsonb;default:'{}'"`
	Importance     float64   `json:"importance" gorm:"default:0.5"`
	AccessCount    int       `json:"access_count" gorm:"default:0"`
	LastAccessedAt *time.Time `json:"last_accessed_at"`
	ExpiresAt      *time.Time `json:"expires_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type JSON map[string]interface{}
type Vector []float32

type Store struct {
	db       *gorm.DB
	embedder provider.Embedder
}

func NewStore(db *gorm.DB, embedder provider.Embedder) *Store {
	return &Store{db: db, embedder: embedder}
}

func (s *Store) StoreMemory(ctx context.Context, orgID uuid.UUID, agentType string, content string, metadata map[string]interface{}) (*MemoryEntry, error) {
	embResp, err := s.embedder.Embed(ctx, &provider.EmbeddingRequest{
		Input: []string{content},
	})
	if err != nil {
		return nil, err
	}

	var embedding Vector
	if len(embResp.Embeddings) > 0 {
		embedding = embResp.Embeddings[0]
	}

	entry := &MemoryEntry{
		OrganizationID: orgID,
		AgentType:      agentType,
		Content:        content,
		Embedding:      embedding,
		Metadata:       JSON(metadata),
		Importance:     0.5,
	}

	if err := s.db.WithContext(ctx).Create(entry).Error; err != nil {
		return nil, err
	}
	return entry, nil
}

func (s *Store) SearchMemories(ctx context.Context, orgID uuid.UUID, query string, limit int) ([]MemoryEntry, error) {
	embResp, err := s.embedder.Embed(ctx, &provider.EmbeddingRequest{
		Input: []string{query},
	})
	if err != nil {
		return nil, err
	}

	if len(embResp.Embeddings) == 0 {
		return nil, nil
	}

	var memories []MemoryEntry
	err = s.db.WithContext(ctx).
		Where("organization_id = ?", orgID).
		Order(gorm.Expr("embedding <=> ?", embResp.Embeddings[0])).
		Limit(limit).
		Find(&memories).Error

	return memories, err
}

func (s *Store) AddNode(ctx context.Context, node *KnowledgeNode) error {
	if node.Name != "" && s.embedder != nil {
		embResp, _ := s.embedder.Embed(ctx, &provider.EmbeddingRequest{
			Input: []string{node.Name},
		})
		if embResp != nil && len(embResp.Embeddings) > 0 {
			node.Embedding = embResp.Embeddings[0]
		}
	}
	return s.db.WithContext(ctx).Create(node).Error
}

func (s *Store) AddEdge(ctx context.Context, edge *KnowledgeEdge) error {
	return s.db.WithContext(ctx).Create(edge).Error
}

func (s *Store) GetNeighbors(ctx context.Context, nodeID uuid.UUID, depth int) ([]KnowledgeNode, []KnowledgeEdge, error) {
	var nodes []KnowledgeNode
	var edges []KnowledgeEdge

	err := s.db.WithContext(ctx).
		Where("id IN (SELECT target_id FROM knowledge_edges WHERE source_id = ?) OR id IN (SELECT source_id FROM knowledge_edges WHERE target_id = ?)", nodeID, nodeID).
		Find(&nodes).Error
	if err != nil {
		return nil, nil, err
	}

	err = s.db.WithContext(ctx).
		Where("source_id = ? OR target_id = ?", nodeID, nodeID).
		Find(&edges).Error

	return nodes, edges, err
}

func (s *Store) SearchNodes(ctx context.Context, orgID uuid.UUID, query string, nodeType NodeType, limit int) ([]KnowledgeNode, error) {
	embResp, err := s.embedder.Embed(ctx, &provider.EmbeddingRequest{
		Input: []string{query},
	})
	if err != nil {
		return nil, err
	}

	if len(embResp.Embeddings) == 0 {
		return nil, nil
	}

	q := s.db.WithContext(ctx).Where("organization_id = ?", orgID)
	if nodeType != "" {
		q = q.Where("node_type = ?", nodeType)
	}

	var nodes []KnowledgeNode
	err = q.Order(gorm.Expr("embedding <=> ?", embResp.Embeddings[0])).
		Limit(limit).
		Find(&nodes).Error

	return nodes, err
}
