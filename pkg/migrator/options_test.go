package migrator

import (
	"database/sql"
	"os"
	"testing"

	"github.com/deahtstroke/gsmt/pkg/dialect"
)

func Test_WithDialect_Success(t *testing.T) {
	tests := map[string]struct {
		Dial dialect.Dialect
	}{
		"postgreSQL dialect": {
			Dial: dialect.Postgres(),
		},
	}

	for testName, params := range tests {
		t.Run(testName, func(t *testing.T) {
			db := sql.DB{}
			fs := os.DirFS("./testdata/")
			migrator, _ := New(&db, WithDialect(params.Dial), WithSchema(fs))

			if migrator.dialect == nil {
				t.Fatalf("Migrator should not have a nil dialect")
			}
		})
	}
}
