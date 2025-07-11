package migrator

import (
	"io/fs"

	"github.com/deahtstroke/gsmt/pkg/dialect"
)

type MigratorOption func(m *Migrator)

// Applies the dialect parameter to the current migrator instance
func WithDialect(dialect dialect.Dialect) MigratorOption {
	return func(m *Migrator) {
		m.dialect = dialect
	}
}

// Applies the schema directory parameter to the current migrator instance
func WithSchema(directory fs.FS) MigratorOption {
	return func(m *Migrator) {
		m.schemaFS = directory
	}
}

// Applies the data directory parameter
func WithData(directory fs.FS) MigratorOption {
	return func(m *Migrator) {
		m.dataFS = directory
	}
}
