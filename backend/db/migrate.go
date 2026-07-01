// db/migrate.go
package db

import (
	_ "embed"
	"log"

	"github.com/jmoiron/sqlx"
)

//go:embed migrations/001_init.sql
var migration001 string

//go:embed migrations/002_pgvector.sql
var migration002 string

func RunMigrations(db *sqlx.DB) {
	for i, sql := range []string{migration001, migration002} {
		if _, err := db.Exec(sql); err != nil {
			log.Fatalf("migration %d failed: %v", i+1, err)
		}
	}
	log.Println("migrations applied")
}
