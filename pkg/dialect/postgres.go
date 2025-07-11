package dialect

import (
	"database/sql"
	"fmt"
)

type PostgreSQLDialect struct{}

func Postgres() Dialect {
	return PostgreSQLDialect{}
}

type TableDDL struct {
	TableName string
	Script    string
}

func SchemaMigrationTableDDL() string {
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

func DataMigrationTableDDL() string {
	return `
	CREATE TABLE IF NOT EXISTS gsmt_data_migrations (
	id SERIAL PRIMARY KEY,
	checksum TEXT NOT NULL,
	applied_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	execution_time_ms BIGINT,
	script_name TEXT NOT NULL
	);
	`
}

func (ps PostgreSQLDialect) SetupMetadataTables(db *sql.DB) error {
	tables := []TableDDL{
		{TableName: "gsmt_migrations", Script: SchemaMigrationTableDDL()},
		{TableName: "gsmt_data_migrations", Script: DataMigrationTableDDL()},
	}

	for _, table := range tables {
		CreateTableIfNotExists(db, table.TableName, table.Script)
	}
	return nil
}

func CreateTableIfNotExists(db *sql.DB, tableName string, ddl string) error {
	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS ( 
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = $1
		)`, tableName).Scan(&exists)
	if err != nil {
		return fmt.Errorf("Error parsing or executing exists query for table [%s]: %v", tableName, err)
	}

	if !exists {
		_, err = db.Exec(ddl)
		if err != nil {
			return fmt.Errorf("Failed to execute ddl for table [%s]: %v", tableName, err)
		}
	}

	return nil
}

func (ps PostgreSQLDialect) Placeholder(i int) string {
	return fmt.Sprintf("$%d", i)
}
