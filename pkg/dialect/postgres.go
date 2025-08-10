package dialect

import (
	"fmt"

	"github.com/deahtstroke/gsmt/pkg/schema"
)

type PostgreSQLDialect struct{}

func Postgres() Dialect {
	return PostgreSQLDialect{}
}

func SchemaMigrationTableDDL() string {
	return fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
		id SERIAL PRIMARY KEY,
		checksum TEXT NOT NULL,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT now(),
		execution_time_ms BIGINT,
		script_name TEXT NOT NULL,
		script_content TEXT
	);
	`, schema.SchemaMigrationsTable)
}

func DataMigrationTableDDL() string {
	return fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS %s (
	id SERIAL PRIMARY KEY,
	checksum TEXT NOT NULL,
	applied_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	execution_time_ms BIGINT,
	script_name TEXT NOT NULL
	);
	`, schema.DataMigrationsTable)
}

func (ps PostgreSQLDialect) GetMetadataTables() map[string]string {
	return map[string]string{
		schema.SchemaMigrationsTable: SchemaMigrationTableDDL(),
		schema.DataMigrationsTable:   DataMigrationTableDDL(),
	}
}

func (ps PostgreSQLDialect) Placeholder(i int) string {
	return fmt.Sprintf("$%d", i)
}
