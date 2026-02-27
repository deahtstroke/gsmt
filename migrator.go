package gsmt

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
)

type Migrator struct {
	store    MetadataStore
	schemaFS fs.FS
	dataFS   fs.FS
}

type MigratorOpts struct {
	Store  MetadataStore
	Schema fs.FS
	Data   fs.FS
}

// Create a new migrator given the options available
func NewMigrator(opts MigratorOpts) (*Migrator, error) {
	if opts.Schema == nil {
		return nil, fmt.Errorf("Schema file system is not declared")
	}

	if opts.Store == nil {
		return nil, fmt.Errorf("Metadata store is nil")
	}

	return &Migrator{
		schemaFS: opts.Schema,
		dataFS:   opts.Data,
		store:    opts.Store,
	}, nil
}

func (m *Migrator) ApplyMigrations(ctx context.Context) error {
	if err := m.ensureMetadataTables(ctx); err != nil {
		return fmt.Errorf("Error ensuring metadata tables: %v", err)
	}

	if err := m.migrateSchema(ctx); err != nil {
		return fmt.Errorf("Error applying schema migrations: %v", err)
	}

	if err := m.migrateData(ctx); err != nil {
		return fmt.Errorf("Error applying data migrations: %v", err)
	}

	return nil
}

func (m *Migrator) ensureMetadataTables(ctx context.Context) error {
	return m.store.SetupMetadataTables(ctx)
}

func (m *Migrator) migrateSchema(ctx context.Context) error {
	appliedMigrations, err := m.store.GetAppliedChecksums(ctx, SchemaMigrationsTable)
	if err != nil {
		return fmt.Errorf("Error fetching applied checksums: %v", err)
	}

	return fs.WalkDir(m.schemaFS, ".", func(path string, d fs.DirEntry, err error) error {
		switch filepath.Ext(path) {
		case ".sql":
			content, err := ReadFileContent(m.schemaFS, path)
			if err != nil {
				return err
			}
			hash := Encode(content)
			if checksum, exists := appliedMigrations[d.Name()]; exists {

				if hash != checksum {
					return fmt.Errorf("Existing schema migration with name %s has a different checksum: Existing: %s. New one: %s", path, checksum, hash)
				}
				return nil
			} else {
				return m.store.RecordSchemaScript(ctx, MigrationScript{
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

func (m *Migrator) migrateData(ctx context.Context) error {
	if m.dataFS == nil {
		return nil
	}

	appliedMigrations, err := m.store.GetAppliedChecksums(ctx, DataMigrationsTable)
	if err != nil {
		return fmt.Errorf("Error fetching applied checksums: %v", err)
	}

	return fs.WalkDir(m.dataFS, ".", func(path string, d fs.DirEntry, err error) error {
		switch filepath.Ext(path) {
		case ".sql":
			content, err := ReadFileContent(m.dataFS, path)
			if err != nil {
				return err
			}
			hash := Encode(content)
			if checksum, exists := appliedMigrations[d.Name()]; exists {
				if hash == checksum {
					return nil
				}
				return fmt.Errorf("Existing data migration with name %s has a different checksum: Exising: %s. New one: %s", path, checksum, hash)
			} else {
				return m.store.RecordDataScript(ctx, MigrationScript{
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
