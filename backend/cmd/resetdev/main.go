// Command resetdev wipes every table in a development database clean —
// organizations, users, roles, tokens, and all domain data — so the very
// next registration starts from a truly empty database. It refuses to run
// against anything but ENVIRONMENT=development.
//
// Two properties matter here and weren't true of the tool this replaced:
//
//  1. Atomicity. Every table is truncated in a single TRUNCATE statement
//     with CASCADE, in one implicit transaction. Postgres either wipes
//     everything or nothing — there is no way for this to die halfway
//     through and leave, say, roles gone but users still present. (The
//     previous version deleted table-by-table in FK-dependency order and
//     missed a couple of tables with NOT NULL references to users —
//     email_verification_tokens, password_reset_tokens — so DELETE FROM
//     users hit a foreign-key violation after user_roles/roles had
//     already been deleted. That's a real incident this tool caused:
//     accounts survived, but every role assignment in the database,
//     including Owner, was already gone.)
//  2. Verification. The tool doesn't just report success because no SQL
//     error was returned — a TRUNCATE against the wrong database (wrong
//     DATABASE_URL, wrong container) "succeeds" too, since truncating
//     zero or the wrong tables isn't a Postgres error. This tool re-counts
//     every table afterward and fails loudly if anything is still there.
package main

import (
	"fmt"
	"log"
	"strings"

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

	var tables []string
	if err := db.Raw(`SELECT tablename FROM pg_tables WHERE schemaname = 'public'`).Scan(&tables).Error; err != nil {
		log.Fatalf("listing tables: %v", err)
	}
	if len(tables) == 0 {
		log.Fatalf("found zero tables in the public schema — this looks like the wrong database, refusing to proceed")
	}

	quoted := make([]string, len(tables))
	for i, t := range tables {
		quoted[i] = `"` + t + `"`
	}

	log.Printf("truncating %d tables: %s", len(tables), strings.Join(tables, ", "))
	stmt := fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE", strings.Join(quoted, ", "))
	if err := db.Exec(stmt).Error; err != nil {
		log.Fatalf("truncate failed (nothing was touched, TRUNCATE is atomic): %v", err)
	}

	var remaining int64
	for _, t := range tables {
		var count int64
		if err := db.Table(t).Count(&count).Error; err != nil {
			log.Fatalf("verifying %q is empty: %v", t, err)
		}
		if count > 0 {
			log.Printf("FAILED VERIFICATION: %q still has %d row(s) after truncate", t, count)
		}
		remaining += count
	}
	if remaining != 0 {
		log.Fatalf("resetdev FAILED: %d row(s) survived the wipe — database is NOT clean, do not treat this as done", remaining)
	}

	log.Println("resetdev complete: every table verified empty — the database is a blank slate, next registration starts fresh")
}
