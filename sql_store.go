package gsmt

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type TableName string

const (
	SchemaTable TableName = "gsmt_schema"
	DataTable   TableName = "gsmt_data"
)

type SQLStore struct {
	db      *sql.DB
	dialect Dialect
}

func NewSQLStore(db *sql.DB, dialect Dialect) Store {
	return &SQLStore{
		db:      db,
		dialect: dialect,
	}
}

func (s *SQLStore) SetupMetadataTables(ctx context.Context) error {
	schemaDDL := s.dialect.CreateDataTable(string(SchemaTable))
	if err := s.createTableIfNotExists(ctx, SchemaTable, schemaDDL); err != nil {
		return err
	}

	dataDDL := s.dialect.CreateDataTable(string(DataTable))
	if err := s.createTableIfNotExists(ctx, DataTable, dataDDL); err != nil {
		return err
	}
	return nil
}

func (s *SQLStore) createTableIfNotExists(ctx context.Context, table TableName, ddl string) error {
	if strings.TrimSpace(ddl) == "" {
		return fmt.Errorf("Passed DDL for table [%s] is empty", string(table))
	}

	var exists bool
	err := s.db.QueryRowContext(ctx, s.dialect.TableExists(), string(table)).
		Scan(&exists)

	if err != nil {
		return fmt.Errorf("Error parsing or executing exists query for table [%s]: %v", table, err)
	}

	if !exists {
		_, err = s.db.ExecContext(ctx, ddl)
		if err != nil {
			return fmt.Errorf("Failed to execute ddl for table [%s]: %v", table, err)
		}
	}
	return nil
}

func (s *SQLStore) GetAppliedChecksums(ctx context.Context, table TableName) (map[string]string, error) {
	rows, err := s.db.QueryContext(ctx, s.dialect.GetChecksums(string(table)))
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

func (s *SQLStore) RecordSchemaScript(ctx context.Context, script MigrationScript) error {
	tx, err := s.db.BeginTx(ctx, nil)
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

func (ms *SQLStore) RecordDataScript(ctx context.Context, script MigrationScript) error {
	tx, err := ms.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("Error beginning transaction for script [%s]: %v", script.Name, err)
	}

	defer tx.Rollback()

	start := time.Now()
	if _, err = tx.ExecContext(ctx, script.Content); err != nil {
		return fmt.Errorf("Error applying data script [%s]: %v", script.Name, err)
	}

	_, err = tx.ExecContext(ctx, ms.dialect.CreateDataTable(string(DataTable)),
		script.Hash, time.Since(start).Milliseconds(), script.Name)
	if err != nil {
		return fmt.Errorf("Error inserting new data script [%s] with checksum [%s]: %v", script.Name, script.Hash, err)
	}

	err = tx.Commit()
	return err
}
