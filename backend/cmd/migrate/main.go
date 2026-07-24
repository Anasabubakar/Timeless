package main

import (
	"flag"
	"log"

	"github.com/sponsoros/backend/internal/config"
	"github.com/sponsoros/backend/internal/database"
)

func main() {
	seed := flag.Bool("seed", false, "seed sample data after migrate")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := database.NewPostgres(cfg)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}

	if err := database.AutoMigrate(db); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	if *seed {
		log.Println("seed flag set: no seed data implemented yet")
	}

	log.Println("migration finished successfully")
}
