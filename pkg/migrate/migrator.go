package migrate

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"gsmt/pkg/dialect"
	"log"
	"os"
	"path"
	"path/filepath"
	"time"
)

type MigrationScript struct {
	Name    string
	Content string
	Hash    string
}

type Migrator struct {
	Db            *sql.DB
	dialect       dialect.Dialect
	rootDirectory string
	scripts       []MigrationScript
}

// Creates a new migrator
// The dialect option is required
func New(db *sql.DB, opts ...MigratorOption) (*Migrator, error) {
	migrator := &Migrator{
		Db:      db,
		scripts: []MigrationScript{},
	}

	// Apply options if there are any
	for _, opt := range opts {
		opt(migrator)
	}

	if migrator.dialect == nil {
		return nil, fmt.Errorf("No dialect selected")
	}

	rootDir, err := findProjectRootDirectory()
	if err != nil {
		return nil, fmt.Errorf("Unable to find project root directory: %v", err)
	}
	rootDir = filepath.Join(rootDir, "migrations", "schema")

	if migrator.rootDirectory != "" {
		rootDir = migrator.rootDirectory
	}

	files, err := os.ReadDir(rootDir)
	if err != nil {
		return nil, fmt.Errorf("Unable to read root directory: %v", err)
	}

	for _, file := range files {
		if ext := path.Ext(file.Name()); ext == ".sql" {
			content, err := os.ReadFile(filepath.Join(rootDir, file.Name()))
			if err != nil {
				return nil, fmt.Errorf("Error reading contents of script: %s: %v", file.Name(), err)
			}

			script := MigrationScript{
				Name:    file.Name(),
				Content: string(content),
				Hash:    encode(string(content)),
			}
			migrator.scripts = append(migrator.scripts, script)
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
	err := m.dialect.EnsureMigrationsTable(m.Db)
	if err != nil {
		panic(fmt.Sprintf("Error ensuring migrations table: %v", err))
	}

	appliedMigrations, err := m.fetchAppliedChecksums(context.Background())
	if err != nil {
		return fmt.Errorf("Error fetching applied checksums: %v", err)
	}

	for _, script := range m.scripts {
		if checksum, exists := appliedMigrations[script.Name]; exists {
			if script.Hash != checksum {
				return fmt.Errorf("An applied migration has a different checksum. Existing: %s | New one: %s", checksum, script.Hash)
			}
			continue
		}

		err = m.applyScript(&script)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *Migrator) applyScript(script *MigrationScript) (err error) {
	tx, err := m.Db.Begin()
	if err != nil {
		return fmt.Errorf("Error beginning transaction for script %s: %v", script.Name, err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	start := time.Now()
	if _, err = tx.Exec(script.Content); err != nil {
		return fmt.Errorf("Error applying migration script %s: %v", script.Name, err)
	}

	insertQuery := fmt.Sprintf(`
			INSERT INTO gsmt_migrations (checksum, execution_time, script_name, script_content)
			VALUES (%s, %s, %s, %s)
			`, m.dialect.Placeholder(1), m.dialect.Placeholder(2), m.dialect.Placeholder(3), m.dialect.Placeholder(4))
	_, err = tx.Exec(insertQuery, script.Hash, time.Since(start).Milliseconds(), script.Name, script.Content)
	if err != nil {
		return fmt.Errorf("Error inserting new gmst_migration with name [%s] and checksum [%s]: %v", script.Name, script.Hash, err)
	}

	err = tx.Commit()
	return err
}

// This function tries to find the directory with the go.mod file
func findProjectRootDirectory() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		log.Panicf("Unable to read current directory: %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}

		dir = parent
	}

	return "", errors.New("go.mod not found in parent directories")
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
