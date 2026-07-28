package database

import (
	"log"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/timeless/backend/internal/models"
)

// AutoMigrate ensures schema exists for all application models.
// Safe to run on every boot; GORM only creates/updates missing tables/columns.
func AutoMigrate(db *gorm.DB) error {
	log.Println("running auto-migration...")

	// Built-in uuid generation (PG13+) lives in pgcrypto on some providers.
	if err := db.Exec(`CREATE EXTENSION IF NOT EXISTS "pgcrypto"`).Error; err != nil {
		log.Printf("warning: could not create pgcrypto extension: %v", err)
	}
	if err := db.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`).Error; err != nil {
		log.Printf("warning: could not create uuid-ossp extension: %v", err)
	}

	// Core schema required for auth, CRM, outreach, and integrations.
	coreModels := []any{
		&models.Organization{},
		&models.User{},
		&models.Role{},
		&models.OAuthAccount{},
		&models.RefreshToken{},
		&models.EmailVerificationToken{},
		&models.PasswordResetToken{},
		&models.Team{},
		&models.TeamMember{},
		&models.Industry{},
		&models.Company{},
		&models.PainPoint{},
		&models.DecisionMaker{},
		&models.Contact{},
		&models.Project{},
		&models.Campaign{},
		&models.Sponsor{},
		&models.Proposal{},
		&models.Activity{},
		&models.Communication{},
		&models.EmailTemplate{},
		&models.Task{},
		&models.OutreachSequence{},
		&models.SequenceStep{},
		&models.Enrollment{},
		&models.AIProvider{},
		&models.AIConversation{},
		&models.AIMessage{},
		&models.Integration{},
		&models.Webhook{},
		&models.WebhookDelivery{},
		&models.SyncRun{},
		&models.Automation{},
		&models.Notification{},
		&models.NotificationPreference{},
		&models.OnboardingState{},
	}

	if err := db.AutoMigrate(coreModels...); err != nil {
		return err
	}

	if err := migrateKnowledgeTables(db); err != nil {
		log.Printf("warning: knowledge/memory migration: %v", err)
	}

	// Ensure many2many join table for user roles exists.
	if err := db.AutoMigrate(&userRole{}); err != nil {
		log.Printf("warning: user_roles join table: %v", err)
	}

	log.Println("auto-migration complete")
	return nil
}

// userRole is the GORM many2many join model for users <-> roles.
type userRole struct {
	UserID uuid.UUID `gorm:"type:uuid;primaryKey"`
	RoleID uuid.UUID `gorm:"type:uuid;primaryKey"`
}

func (userRole) TableName() string {
	return "user_roles"
}

// Lite models omit vector(1536) columns so Neon/Render work without pgvector.
type knowledgeNodeLite struct {
	models.Base
	OrganizationID uuid.UUID      `gorm:"type:uuid;not null;index"`
	NodeType       string         `gorm:"size:50;not null;index"`
	EntityID       *uuid.UUID     `gorm:"type:uuid"`
	Label          string         `gorm:"size:500;not null"`
	Properties     datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'"`
}

func (knowledgeNodeLite) TableName() string { return "knowledge_nodes" }

type knowledgeEdgeLite struct {
	models.Base
	OrganizationID uuid.UUID      `gorm:"type:uuid;not null;index"`
	SourceNodeID   uuid.UUID      `gorm:"type:uuid;not null;index"`
	TargetNodeID   uuid.UUID      `gorm:"type:uuid;not null;index"`
	RelationType   string         `gorm:"size:100;not null"`
	Weight         float64        `gorm:"not null;default:1.0"`
	Properties     datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'"`
}

func (knowledgeEdgeLite) TableName() string { return "knowledge_edges" }

type aiMemoryLite struct {
	models.Base
	OrganizationID uuid.UUID      `gorm:"type:uuid;not null;index"`
	MemoryType     string         `gorm:"size:50;not null"`
	Content        string         `gorm:"type:text;not null"`
	Importance     float64        `gorm:"not null;default:0.5"`
	AccessCount    int            `gorm:"not null;default:0"`
	EntityType     *string        `gorm:"size:50"`
	EntityID       *uuid.UUID     `gorm:"type:uuid"`
	Metadata       datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'"`
}

func (aiMemoryLite) TableName() string { return "ai_memories" }

func migrateKnowledgeTables(db *gorm.DB) error {
	vectorOK := db.Exec(`CREATE EXTENSION IF NOT EXISTS "vector"`).Error == nil
	if vectorOK {
		if err := db.AutoMigrate(
			&models.KnowledgeNode{},
			&models.KnowledgeEdge{},
			&models.AIMemory{},
		); err == nil {
			return nil
		} else {
			log.Printf("warning: full vector migrate failed (%v); falling back without embeddings", err)
		}
	} else {
		log.Printf("warning: pgvector unavailable; creating knowledge tables without embeddings")
	}

	return db.AutoMigrate(
		&knowledgeNodeLite{},
		&knowledgeEdgeLite{},
		&aiMemoryLite{},
	)
}
