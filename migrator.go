package gsmt

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"path/filepath"
)

type migrator struct {
	migrationFS fs.FS
}

func ApplyMigrations(ctx context.Context, db *sql.DB) error {
	if err := s.SetupMetadataTables(ctx, db); err != nil {
		return fmt.Errorf("Error ensuring metadata tables are created: %v", err)
	}

	err := m.migrateSchema(ctx, db)
	return err
}

func (m *migrator) migrateSchema(ctx context.Context, db *sql.DB) error {
	appliedMigrations, err := s.GetAppliedChecksums(ctx, db, SchemaTable)
	if err != nil {
		return fmt.Errorf("Error fetching applied checksums: %v", err)
	}

	return fs.WalkDir(m.migrationFS, ".", func(path string, d fs.DirEntry, err error) error {
		switch filepath.Ext(path) {
		case ".sql":
			content, err := ReadFileContent(m.migrationFS, path)
			if err != nil {
				return err
			}
			hash := Hash(content)
			if checksum, exists := appliedMigrations[d.Name()]; exists {

				if hash != checksum {
					return fmt.Errorf("Existing schema migration with name %s has a different checksum: Existing: %s. New one: %s", path, checksum, hash)
				}
				return nil
			} else {
				return s.RecordSchemaScript(ctx, db, MigrationScript{
					Name:    d.Name(),
					Content: content,
					Hash:    hash,
				})
			}
		default:
			return nil
		}
	})
}
