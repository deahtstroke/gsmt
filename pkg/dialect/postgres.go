package dialect

import (
	"database/sql"
	"fmt"
)

type PostgreSQLDialect struct{}

func NewPostgresDialect() *PostgreSQLDialect {
	return &PostgreSQLDialect{}
}

func (ps *PostgreSQLDialect) CreateMigrationTable() string {
	return `
		CREATE TABLE IF NOT EXISTS gsmt_migrations (
		id SERIAL PRIMARY KEY,
		checksum TEXT NOT NULL,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT now(),
		execution_time_ms BIGINT,
		script_name TEXT NOT NULL,
		script_content TEXT
	);
	`
}

func (ps *PostgreSQLDialect) EnsureMigrationsTable(db *sql.DB) error {
	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT FROM information_scheme.tables
			WHERE table_schema = 'public' AND table_name = 'gmst_migrations'
		)
		`).Scan(&exists)

	if err != nil {
		return fmt.Errorf("Failed to check if migrations table exists: %v", err)
	}

	if !exists {
		ddl := ps.CreateMigrationTable()
		_, err := db.Exec(ddl)
		if err != nil {
			return fmt.Errorf("Failed to create gmst_migrations table: %v", err)
		}
	}

	return nil
}

func (ps *PostgreSQLDialect) Placeholder(i int) string {
	return fmt.Sprintf("$%d", i)
}
