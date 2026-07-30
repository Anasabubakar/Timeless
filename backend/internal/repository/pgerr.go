package repository

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// pgUniqueViolation is the Postgres SQLSTATE for a unique constraint
// violation (23505) — https://www.postgresql.org/docs/current/errcodes-appendix.html
const pgUniqueViolation = "23505"

// IsUniqueViolation reports whether err is a Postgres unique constraint
// violation, optionally narrowed to a specific constraint name (pass ""
// to match any). Callers use this to distinguish "a concurrent request
// beat us to this exact row" — recoverable by retrying with a different
// candidate value — from every other kind of database failure, which
// isn't.
func IsUniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != pgUniqueViolation {
		return false
	}
	return constraint == "" || pgErr.ConstraintName == constraint
}
