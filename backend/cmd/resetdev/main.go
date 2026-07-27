// Command resetdev clears user and onboarding data from a development
// database so every account goes through the new onboarding flow from
// scratch. It refuses to run against anything but ENVIRONMENT=development
// and never touches organizations or domain data (companies, sponsors,
// etc.) — only accounts and onboarding state are wiped.
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

	tables := []string{
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

	log.Println("resetdev complete: all users and onboarding state removed, organizations and domain data kept")
}
