package gsmt

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/deahtstroke/gsmt/dialect"
)

type TableName string

const (
	SchemaTable TableName = "gsmt_schema"
	DataTable   TableName = "gsmt_data"
)

type SQLStore struct {
	dialect dialect.Dialect
}

func NewSQLStore(dialect dialect.Dialect) Store {
	return &SQLStore{
		dialect: dialect,
	}
}

func (s *SQLStore) SetupMetadataTables(ctx context.Context, db *sql.DB) error {
	schemaDDL := s.dialect.CreateMetadataTable(string(SchemaTable))
	if err := s.createTableIfNotExists(ctx, db, SchemaTable, schemaDDL); err != nil {
		return err
	}
	return nil
}

func (s *SQLStore) createTableIfNotExists(ctx context.Context, db *sql.DB, table TableName, ddl string) error {
	if strings.TrimSpace(ddl) == "" {
		return fmt.Errorf("Passed DDL for table [%s] is empty", string(table))
	}

	var exists bool
	err := db.QueryRowContext(ctx, s.dialect.TableExists(), string(table)).
		Scan(&exists)

	if err != nil {
		return fmt.Errorf("Error parsing or executing exists query for table [%s]: %v", table, err)
	}

	if !exists {
		_, err = db.ExecContext(ctx, ddl)
		if err != nil {
			return fmt.Errorf("Failed to execute ddl for table [%s]: %v", table, err)
		}
	}
	return nil
}

func (s *SQLStore) GetAppliedChecksums(ctx context.Context, db *sql.DB, table TableName) (map[string]string, error) {
	rows, err := db.QueryContext(ctx, s.dialect.GetChecksums(string(table)))
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

func (s *SQLStore) RecordSchemaScript(ctx context.Context, db *sql.DB, script MigrationScript) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("Error beginning transaction for script [%s]: %v", script.Name, err)
	}

	defer tx.Rollback()

	start := time.Now()
	if _, err = tx.ExecContext(ctx, script.Content); err != nil {
		return fmt.Errorf("Error applying migration script [%s]: %v", script.Name, err)
	}

	_, err = tx.ExecContext(ctx, s.dialect.InsertMetadata(string(SchemaTable)), script.Hash, time.Since(start).Milliseconds(), script.Name, script.Content)
	if err != nil {
		return fmt.Errorf("Error inserting new schema script [%s] with checksum [%s]: %v", script.Name, script.Hash, err)
	}

	err = tx.Commit()
	return err
}
