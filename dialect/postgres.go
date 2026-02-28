package dialect

import (
	"fmt"
	"github.com/deahtstroke/gsmt"
)

type Postgres struct{}

func PostgresDialect() gsmt.Dialect {
	return &Postgres{}
}

// Compiler check
var _ gsmt.Dialect = (*Postgres)(nil)

func (p *Postgres) CreateMetadataTable(name string) string {
	return fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id SERIAL PRIMARY KEY,
			checksum TEXT NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			execution_time_ms BIGINT,
			script_name TEXT NOT NULL,
			script_content TEXT
		);
	`, name)
}

func (p *Postgres) CreateDataTable(name string) string {
	return fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS %s (
	id SERIAL PRIMARY KEY,
	checksum TEXT NOT NULL,
	applied_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	execution_time_ms BIGINT,
	script_name TEXT NOT NULL
	);
	`, name)
}

func (p *Postgres) InsertMetadata(name string) string {
	return fmt.Sprintf(`
		INSERT INTO %s (checksum, execution_time_ms, script_name, script_content)
		VALUES ($1, $2, $3, $4);
	`, name)
}

func (p *Postgres) InsertData(name string) string {
	return fmt.Sprintf(`
	INSERT INTO %s (checksum, execution_ms, script_name, applied_at)
	VALUES ($1, $2, $3, $4)
	`, name)
}

func (p *Postgres) GetChecksums(name string) string {
	return fmt.Sprintf(`SELECT script_name, checksum FROM %s`, name)
}

func (p *Postgres) TableExists() string {
	return `
		SELECT EXISTS ( 
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = $1
		)
	`
}
