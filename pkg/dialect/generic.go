package dialect

import (
	"database/sql"
)

type Dialect interface {
	Placeholder(i int) string
	CreateMigrationTable() string
	EnsureMigrationsTable(db *sql.DB) error
}
