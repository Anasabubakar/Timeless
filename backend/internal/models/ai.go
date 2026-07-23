package models

import (
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type AIProvider struct {
	Base
	OrganizationID *uuid.UUID     `gorm:"type:uuid" json:"organization_id,omitempty"`
	Name           string         `gorm:"size:100;not null" json:"name"`
	Slug           string         `gorm:"size:50;not null" json:"slug"`
	APIKeyEnc      *string        `gorm:"type:text" json:"-"`
	BaseURL        *string        `gorm:"type:text" json:"base_url,omitempty"`
	Models         datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"models"`
	IsEnabled      bool           `gorm:"not null;default:true" json:"is_enabled"`
	IsGlobal       bool           `gorm:"not null;default:false" json:"is_global"`
	RateLimit      *int           `json:"rate_limit,omitempty"`
	Config         datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"config"`
}

func (AIProvider) TableName() string {
	return "ai_providers"
}

type AIConversation struct {
	Base
	OrganizationID uuid.UUID      `gorm:"type:uuid;not null;index" json:"organization_id"`
	UserID         uuid.UUID      `gorm:"type:uuid;not null;index" json:"user_id"`
	Title          *string        `gorm:"size:255" json:"title,omitempty"`
	AgentType      string         `gorm:"size:50;not null" json:"agent_type"`
	Status         string         `gorm:"size:20;not null;default:active" json:"status"`
	Context        datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"context"`
	Messages       []AIMessage    `gorm:"foreignKey:ConversationID" json:"messages,omitempty"`

	Organization Organization `gorm:"foreignKey:OrganizationID" json:"-"`
	User         User         `gorm:"foreignKey:UserID" json:"-"`
}

func (AIConversation) TableName() string {
	return "ai_conversations"
}

type AIMessage struct {
	Base
	ConversationID uuid.UUID      `gorm:"type:uuid;not null;index" json:"conversation_id"`
	Role           string         `gorm:"size:20;not null" json:"role"`
	Content        string         `gorm:"type:text;not null" json:"content"`
	ToolCalls      datatypes.JSON `gorm:"type:jsonb" json:"tool_calls,omitempty"`
	ToolResults    datatypes.JSON `gorm:"type:jsonb" json:"tool_results,omitempty"`
	TokensUsed     *int           `json:"tokens_used,omitempty"`
	Model          *string        `gorm:"size:100" json:"model,omitempty"`
	LatencyMs      *int           `json:"latency_ms,omitempty"`

	Conversation AIConversation `gorm:"foreignKey:ConversationID" json:"-"`
}

func (AIMessage) TableName() string {
	return "ai_messages"
}

type KnowledgeNode struct {
	Base
	OrganizationID uuid.UUID      `gorm:"type:uuid;not null;index" json:"organization_id"`
	NodeType       string         `gorm:"size:50;not null;index" json:"node_type"`
	EntityID       *uuid.UUID     `gorm:"type:uuid" json:"entity_id,omitempty"`
	Label          string         `gorm:"size:500;not null" json:"label"`
	Properties     datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"properties"`
	Embedding      []byte         `gorm:"type:vector(1536)" json:"-"`

	Organization Organization    `gorm:"foreignKey:OrganizationID" json:"-"`
	OutEdges     []KnowledgeEdge `gorm:"foreignKey:SourceNodeID" json:"out_edges,omitempty"`
	InEdges      []KnowledgeEdge `gorm:"foreignKey:TargetNodeID" json:"in_edges,omitempty"`
}

func (KnowledgeNode) TableName() string {
	return "knowledge_nodes"
}

type KnowledgeEdge struct {
	Base
	OrganizationID uuid.UUID      `gorm:"type:uuid;not null;index" json:"organization_id"`
	SourceNodeID   uuid.UUID      `gorm:"type:uuid;not null;index" json:"source_node_id"`
	TargetNodeID   uuid.UUID      `gorm:"type:uuid;not null;index" json:"target_node_id"`
	RelationType   string         `gorm:"size:100;not null" json:"relation_type"`
	Weight         float64        `gorm:"not null;default:1.0" json:"weight"`
	Properties     datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"properties"`

	Organization Organization  `gorm:"foreignKey:OrganizationID" json:"-"`
	SourceNode   KnowledgeNode `gorm:"foreignKey:SourceNodeID" json:"source_node,omitempty"`
	TargetNode   KnowledgeNode `gorm:"foreignKey:TargetNodeID" json:"target_node,omitempty"`
}

func (KnowledgeEdge) TableName() string {
	return "knowledge_edges"
}

type AIMemory struct {
	Base
	OrganizationID uuid.UUID      `gorm:"type:uuid;not null;index" json:"organization_id"`
	MemoryType     string         `gorm:"size:50;not null" json:"memory_type"`
	Content        string         `gorm:"type:text;not null" json:"content"`
	Embedding      []byte         `gorm:"type:vector(1536)" json:"-"`
	Importance     float64        `gorm:"not null;default:0.5" json:"importance"`
	AccessCount    int            `gorm:"not null;default:0" json:"access_count"`
	LastAccessed   *interface{}   `gorm:"-" json:"last_accessed,omitempty"`
	EntityType     *string        `gorm:"size:50" json:"entity_type,omitempty"`
	EntityID       *uuid.UUID     `gorm:"type:uuid" json:"entity_id,omitempty"`
	Metadata       datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"metadata"`

	Organization Organization `gorm:"foreignKey:OrganizationID" json:"-"`
}

func (AIMemory) TableName() string {
	return "ai_memories"
}
