package dialect

import (
	"testing"

	"github.com/deahtstroke/gsmt/pkg/schema"
)

func Test_GetMetadataTables(t *testing.T) {
	dialect := Postgres()
	metadataTables := dialect.GetMetadataTables()

	if metadataTables[schema.DataMigrationsTable] != DataMigrationTableDDL() {
		t.Error("Data migration table does not match its DDL")
	}

	if metadataTables[schema.SchemaMigrationsTable] != SchemaMigrationTableDDL() {
		t.Error("Schema migration table does not match its DDL")
	}
}
