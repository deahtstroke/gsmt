package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/deahtstroke/gsmt/pkg/data"
	"github.com/deahtstroke/gsmt/pkg/dialect"
)

type SQLMigrationStore struct {
	db      *sql.DB
	dialect dialect.Dialect
}

func NewSQLStore(db *sql.DB, dialect dialect.Dialect) SQLMigrationStore {
	return SQLMigrationStore{
		db:      db,
		dialect: dialect,
	}
}

func (ms *SQLMigrationStore) SetupMetadataTables(ctx context.Context) error {
	for name, dll := range ms.dialect.GetMetadataTables() {
		ms.createTableIfNotExists(ctx, name, dll)
	}
	return nil
}

func (ms *SQLMigrationStore) createTableIfNotExists(ctx context.Context, tableName string, ddl string) error {
	if strings.TrimSpace(ddl) == "" {
		return fmt.Errorf("Passed DDL for table [%s] is empty", tableName)
	}
	var exists bool
	err := ms.db.QueryRowContext(ctx, `
		SELECT EXISTS ( 
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = $1
		)`, tableName).Scan(&exists)
	if err != nil {
		return fmt.Errorf("Error parsing or executing exists query for table [%s]: %v", tableName, err)
	}

	if !exists {
		_, err = ms.db.ExecContext(ctx, ddl)
		if err != nil {
			return fmt.Errorf("Failed to execute ddl for table [%s]: %v", tableName, err)
		}
	}
	return nil
}

func (ms *SQLMigrationStore) GetAppliedChecksums(ctx context.Context, table string) (map[string]string, error) {
	query := fmt.Sprintf(`SELECT script_name, checksum FROM %s`, table)
	rows, err := ms.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	applied := make(map[string]string)

	for rows.Next() {
		var name, checksum string
		if err := rows.Scan(&name, &checksum); err != nil {
			return nil, err
		}

		applied[name] = checksum
	}
	return applied, nil
}

func (ms *SQLMigrationStore) RecordSchemaScript(ctx context.Context, script data.MigrationScript) error {
	tx, err := ms.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("Error beginning transaction for script [%s]: %v", script.Name, err)
	}

	defer tx.Rollback()

	start := time.Now()
	if _, err = tx.ExecContext(ctx, script.Content); err != nil {
		return fmt.Errorf("Error applying migration script [%s]: %v", script.Name, err)
	}

	recordScriptQuery := fmt.Sprintf(`
			INSERT INTO %s (checksum, execution_time_ms, script_name, script_content)
			VALUES (%s, %s, %s, %s)
		`, data.SchemaMigrationsTable, ms.dialect.Placeholder(1), ms.dialect.Placeholder(2), ms.dialect.Placeholder(3), ms.dialect.Placeholder(4))

	_, err = tx.ExecContext(ctx, recordScriptQuery, script.Hash, time.Since(start).Milliseconds(), script.Name, script.Content)
	if err != nil {
		return fmt.Errorf("Error inserting new schema script [%s] with checksum [%s]: %v", script.Name, script.Hash, err)
	}

	err = tx.Commit()
	return err
}

func (ms *SQLMigrationStore) RecordDataScript(ctx context.Context, script data.MigrationScript) error {
	tx, err := ms.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("Error beginning transaction for script [%s]: %v", script.Name, err)
	}

	defer tx.Rollback()

	start := time.Now()
	if _, err = tx.ExecContext(ctx, script.Content); err != nil {
		return fmt.Errorf("Error applying data script [%s]: %v", script.Name, err)
	}

	recordScriptQuery := fmt.Sprintf(`
		INSERT INTO %s (checksum, execution_time_ms, script_name)
		VALUES (%s, %s, %s)
		`, data.DataMigrationsTable, ms.dialect.Placeholder(1), ms.dialect.Placeholder(2), ms.dialect.Placeholder(3))

	_, err = tx.ExecContext(ctx, recordScriptQuery, script.Hash, time.Since(start).Milliseconds(), script.Name)
	if err != nil {
		return fmt.Errorf("Error inserting new data script [%s] with checksum [%s]: %v", script.Name, script.Hash, err)
	}

	err = tx.Commit()
	return err
}
