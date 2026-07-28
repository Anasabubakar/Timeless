// Command resetdev clears user and onboarding data from a development
// database so every account goes through the new onboarding flow from
// scratch. It refuses to run against anything but ENVIRONMENT=development
// and never touches organizations or domain data (companies, sponsors,
// etc.) — only accounts and onboarding/personal state are wiped. Nullable
// references to users (created_by, assigned_to, etc.) are cleared rather
// than cascading the delete, so domain records survive.
package main

import (
	"log"

	"github.com/timeless/backend/internal/config"
	"github.com/timeless/backend/internal/database"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	if !cfg.IsDevelopment() {
		log.Fatalf("refusing to run: ENVIRONMENT=%q, this command only runs when ENVIRONMENT=development", cfg.Environment)
	}

	db, err := database.NewPostgres(cfg)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}

	nullOuts := []struct{ table, column string }{
		{"integrations", "installed_by"},
		{"automations", "created_by"},
		{"activities", "user_id"},
		{"communications", "sent_by"},
		{"tasks", "assigned_to"},
		{"tasks", "created_by"},
		{"sponsors", "assigned_to"},
		{"proposals", "created_by"},
		{"campaigns", "created_by"},
		{"projects", "created_by"},
		{"outreach_sequences", "created_by"},
	}
	for _, n := range nullOuts {
		if err := db.Exec("UPDATE " + n.table + " SET " + n.column + " = NULL WHERE " + n.column + " IS NOT NULL").Error; err != nil {
			log.Fatalf("nulling %s.%s: %v", n.table, n.column, err)
		}
		log.Printf("cleared reference %s.%s", n.table, n.column)
	}

	tables := []string{
		"ai_messages",
		"ai_conversations",
		"team_members",
		"notifications",
		"notification_preferences",
		"onboarding_states",
		"refresh_tokens",
		"oauth_accounts",
		"user_roles",
		"roles",
		"users",
	}

	for _, table := range tables {
		if err := db.Exec("DELETE FROM " + table).Error; err != nil {
			log.Fatalf("clearing %s: %v", table, err)
		}
		log.Printf("cleared %s", table)
	}

	log.Println("resetdev complete: all users and onboarding/personal state removed, organizations and domain data kept")
}
