package dialect

import (
	"fmt"
)

type Sqlite3 struct{}

var _ Dialect = (*Sqlite3)(nil)

func NewSqlite3() *Sqlite3 {
	return &Sqlite3{}
}

func (s *Sqlite3) CreateMetadataTable(name string) string {
	return fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS "%s" (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			checksum TEXT NOT NULL,
			applied_at INTEGER DEFAULT (strftime('%%s', 'now')),
			execution_time_ms INTEGER,
			script_name TEXT NOT NULL,
			script_content TEXT
		);
	`, name)
}

// GetChecksums implements [gsmt.Dialect].
func (s *Sqlite3) GetChecksums(name string) string {
	return fmt.Sprintf(`
	SELECT script_name, checksum FROM %s;
	`, name)
}

// InsertMetadata implements [gsmt.Dialect].
func (s *Sqlite3) InsertMetadata(name string) string {
	return fmt.Sprintf(`
		INSERT INTO %s (checksum, execution_time_ms, script_name, script_content)
		VALUES (?, ?, ?, ?);
	`, name)
}

// TableExists implements [gsmt.Dialect].
func (s *Sqlite3) TableExists() string {
	return `
		SELECT EXISTS (
			SELECT 1
			FROM sqlite_master
			WHERE type = 'table'
				AND name = ?
		);
	`
}
