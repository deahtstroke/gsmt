package migrate

import (
	"github.com/deahtstroke/gsmt/pkg/dialect"
)

type MigratorOption func(m *Migrator)

// Applies the dialect parameter to the current migrator instance
func WithDialect(dialect dialect.Dialect) MigratorOption {
	return func(m *Migrator) {
		m.dialect = dialect
	}
}

// Applies the directory parameter to the current migrator instance
func WithRootDirectory(directory string) MigratorOption {
	return func(m *Migrator) {
		m.rootDirectory = directory
	}
}
