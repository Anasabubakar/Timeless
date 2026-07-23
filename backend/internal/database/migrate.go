package database

import (
	"log"

	"gorm.io/gorm"

	"github.com/sponsoros/backend/internal/models"
)

func AutoMigrate(db *gorm.DB) error {
	log.Println("running auto-migration...")

	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"").Error; err != nil {
		log.Printf("warning: could not create uuid-ossp extension: %v", err)
	}
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS \"vector\"").Error; err != nil {
		log.Printf("warning: could not create vector extension: %v", err)
	}

	return db.AutoMigrate(
		&models.Organization{},
		&models.User{},
		&models.Company{},
		&models.Contact{},
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
		&models.AIMemory{},
		&models.KnowledgeNode{},
		&models.KnowledgeEdge{},
		&models.Integration{},
		&models.Webhook{},
		&models.WebhookDelivery{},
		&models.Automation{},
	)
}
