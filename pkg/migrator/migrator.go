package migrator

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"time"

	"github.com/deahtstroke/gsmt/pkg/dialect"
)

type MigrationScript struct {
	Name    string
	Content string
	Hash    string
}

type Migrator struct {
	Db            *sql.DB
	dialect       dialect.Dialect
	schemaFS      fs.FS
	dataFS        fs.FS
	schemaScripts []MigrationScript
	dataScripts   []MigrationScript
}

// Creates a new migrator
// Required options are:
// 1. Dialect
// 2. Schema directory (Embedded or non-embedded)
func New(db *sql.DB, opts ...MigratorOption) (*Migrator, error) {
	migrator := &Migrator{
		Db:            db,
		schemaScripts: []MigrationScript{},
	}

	// Apply options if there are any
	for _, opt := range opts {
		opt(migrator)
	}

	if migrator.dialect == nil {
		return nil, fmt.Errorf("No dialect selected")
	}

	if migrator.schemaFS == nil {
		return nil, fmt.Errorf("Schema directory is not declared")
	}

	err := fs.WalkDir(migrator.schemaFS, ".", func(path string, d fs.DirEntry, err error) error {
		if filepath.Ext(path) == ".sql" {
			file, err := migrator.schemaFS.Open(path)
			if err != nil {
				return fmt.Errorf("Error opening file [%s]: %v", d.Name(), err)
			}

			defer file.Close()

			bytes, err := io.ReadAll(file)
			if err != nil {
				return fmt.Errorf("Error reading contents of file [%s]: %v", d.Name(), err)
			}

			content := string(bytes)
			script := MigrationScript{
				Name:    d.Name(),
				Content: content,
				Hash:    encode(content),
			}
			migrator.schemaScripts = append(migrator.schemaScripts, script)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Data portion
	if migrator.dataFS != nil {
		err = fs.WalkDir(migrator.dataFS, ".", func(path string, d fs.DirEntry, err error) error {
			if filepath.Ext(path) == ".sql" {
				file, err := migrator.dataFS.Open(path)
				if err != nil {
					return fmt.Errorf("Error while opening data script file [%s]: %v", d.Name(), err)
				}

				defer file.Close()
				bytes, err := io.ReadAll(file)

				if err != nil {
					return fmt.Errorf("Error reading contents of data script file [%s]: %v", d.Name(), err)
				}

				content := string(bytes)
				script := MigrationScript{
					Name:    d.Name(),
					Content: content,
					Hash:    encode(content),
				}
				migrator.dataScripts = append(migrator.dataScripts, script)
			}
			return nil
		})

		if err != nil {
			return nil, err
		}
	}

	return migrator, nil
}

// Applies all migrations that are found in the migrator.RootDirectory field
// Additionally, it ensures that the standard gsmt_migrations table exists before applying the scripts
//
// ApplyMigrations() also encapsulates each script in a transaction so that if any error happens
// it'll be rolled back
func (m *Migrator) ApplyMigrations() error {
	err := m.dialect.SetupMetadataTables(m.Db)
	if err != nil {
		panic(fmt.Sprintf("Error ensuring migrations table: %v", err))
	}

	appliedMigrations, err := m.fetchAppliedChecksums(context.Background())
	if err != nil {
		return fmt.Errorf("Error fetching applied checksums: %v", err)
	}

	for _, script := range m.schemaScripts {
		if checksum, exists := appliedMigrations[script.Name]; exists {
			if script.Hash != checksum {
				return fmt.Errorf("An applied migration has a different checksum. Existing: %s | New one: %s", checksum, script.Hash)
			}
			continue
		}

		err = script.Apply(m.Db, m.dialect)
		if err != nil {
			return err
		}
	}

	return nil
}

func (ms *MigrationScript) Apply(db *sql.DB, dialect dialect.Dialect) (err error) {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("Error beginning transaction for script %s: %v", ms.Name, err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	start := time.Now()
	if _, err = tx.Exec(ms.Content); err != nil {
		return fmt.Errorf("Error applying migration script %s: %v", ms.Name, err)
	}

	insertQuery := fmt.Sprintf(`
			INSERT INTO gsmt_migrations (checksum, execution_time_ms, script_name, script_content)
			VALUES (%s, %s, %s, %s)
			`, dialect.Placeholder(1), dialect.Placeholder(2), dialect.Placeholder(3), dialect.Placeholder(4))
	_, err = tx.Exec(insertQuery, ms.Hash, time.Since(start).Milliseconds(), ms.Name, ms.Content)
	if err != nil {
		return fmt.Errorf("Error inserting new gmst_migration with name [%s] and checksum [%s]: %v", ms.Name, ms.Hash, err)
	}

	err = tx.Commit()
	return err
}

func encode(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// Fetches all applied checksums from the gsmt_migrations table and groups them
// by script name
func (m *Migrator) fetchAppliedChecksums(ctx context.Context) (map[string]string, error) {
	rows, err := m.Db.QueryContext(ctx, `SELECT script_name, checksum FROM gsmt_migrations`)
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
