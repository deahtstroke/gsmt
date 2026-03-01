package gsmt

import (
	"context"
	"database/sql"
)

// Defines a generic store struct that is able to do migration operations
type Store interface {
	SetupMetadataTables(context.Context, *sql.DB) error
	GetAppliedChecksums(context.Context, *sql.DB, TableName) (map[string]string, error)
	RecordSchemaScript(context.Context, *sql.DB, MigrationScript) error
}
