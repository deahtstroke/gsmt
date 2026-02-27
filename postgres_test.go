package gsmt

import (
	"testing"
)

func Test_GetMetadataTables(t *testing.T) {
	dialect := Postgres()
	metadataTables := dialect.GetMetadataTables()

	if metadataTables[DataMigrationsTable] != DataMigrationTableDDL() {
		t.Error("Data migration table does not match its DDL")
	}

	if metadataTables[SchemaMigrationsTable] != SchemaMigrationTableDDL() {
		t.Error("Schema migration table does not match its DDL")
	}
}
