package dialect

import (
	"testing"

	"github.com/deahtstroke/gsmt/pkg/data"
)

func Test_GetMetadataTables(t *testing.T) {
	dialect := Postgres()
	metadataTables := dialect.GetMetadataTables()

	if metadataTables[data.DataMigrationsTable] != DataMigrationTableDDL() {
		t.Error("Data migration table does not match its DDL")
	}

	if metadataTables[data.SchemaMigrationsTable] != SchemaMigrationTableDDL() {
		t.Error("Schema migration table does not match its DDL")
	}
}
