package gsmt

import (
	"io/fs"
	"strings"

	"github.com/deahtstroke/gsmt/dialect"
)

// This instance is the main migrator that is used when executing
// functions such as migrate()
var m migrator = migrator{}

// Global store instance that executes the actual SQL depending on the
// chosen dialect. Default SQL dialect is Postgres
var s Store = &SQLStore{
	dialect: dialect.NewPostgres(),
}

// Sets the dialect to use for the SQL store object that interacts
// with the database
func SetDialect(dbDialect string) error {
	var d dialect.Dialect
	switch strings.ToLower(dbDialect) {
	case "postgres":
		d = dialect.NewPostgres()
	case "sqlite3":
		d = dialect.NewSqlite3()
	}

	s = NewSQLStore(d)
	return nil
}

// Sets the schema file system for the migrator to find
// all the SQL scripts to execute
func SetMigrationFS(fs fs.FS) {
	m.migrationFS = fs
}
