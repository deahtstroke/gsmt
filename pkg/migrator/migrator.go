package migrator

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/deahtstroke/gsmt/pkg/data"
	"github.com/deahtstroke/gsmt/pkg/store"
	"github.com/deahtstroke/gsmt/pkg/utils"
)

type Migrator struct {
	store    store.MetadataStore
	schemaFS fs.FS
	dataFS   fs.FS
}

type MigratorOpts struct {
	store    store.MetadataStore
	schemaFS fs.FS
	dataFS   fs.FS
}

func NewMigrator(opts MigratorOpts) (*Migrator, error) {
	if opts.schemaFS == nil {
		return nil, fmt.Errorf("Schema file system is not declared")
	}

	if opts.store == nil {
		return nil, fmt.Errorf("Metadata store is nil")
	}

	return &Migrator{
		schemaFS: opts.schemaFS,
		dataFS:   opts.dataFS,
		store:    opts.store,
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
	appliedMigrations, err := m.store.GetAppliedChecksums(ctx, data.SchemaMigrationsTable)
	if err != nil {
		return fmt.Errorf("Error fetching applied checksums: %v", err)
	}

	return fs.WalkDir(m.schemaFS, ".", func(path string, d fs.DirEntry, err error) error {
		if filepath.Ext(path) == ".sql" {
			content, err := utils.ReadFileContent(m.schemaFS, path)
			if err != nil {
				return err
			}
			hash := utils.Encode(content)
			if checksum, exists := appliedMigrations[d.Name()]; exists {

				if hash != checksum {
					return fmt.Errorf("Schema migration with name %s has a different checksum: Existing: %s. New one: %s", path, checksum, hash)
				}
				return nil
			} else {
				return m.store.RecordSchemaScript(ctx, data.MigrationScript{
					Name:    d.Name(),
					Content: content,
					Hash:    hash,
				})
			}
		}
		return nil
	})
}

func (m *Migrator) migrateData(ctx context.Context) error {
	appliedMigrations, err := m.store.GetAppliedChecksums(ctx, data.DataMigrationsTable)
	if err != nil {
		return fmt.Errorf("Error fetching applied checksums: %v", err)
	}

	return fs.WalkDir(m.dataFS, ".", func(path string, d fs.DirEntry, err error) error {
		if filepath.Ext(path) == ".sql" {
			content, err := utils.ReadFileContent(m.dataFS, path)
			if err != nil {
				return err
			}
			hash := utils.Encode(content)
			if checksum, exists := appliedMigrations[d.Name()]; exists {
				if hash != checksum {
					return fmt.Errorf("Data migrations with name %s has a different checksum: Exising: %s. New one: %s", path, checksum, hash)
				}
				return nil
			} else {
				return m.store.RecordDataScript(ctx, data.MigrationScript{
					Name:    d.Name(),
					Content: content,
					Hash:    hash,
				})
			}
		}
		return nil
	})
}
