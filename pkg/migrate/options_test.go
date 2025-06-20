package migrate

import (
	"database/sql"
	"github.com/deahtstroke/gsmt/pkg/dialect"
	"testing"
)

func Test_WithDialect_Success(t *testing.T) {
	tests := map[string]struct {
		Dial dialect.Dialect
	}{
		"postgreSQL dialect": {
			Dial: dialect.NewPostgresDialect(),
		},
	}

	for testName, params := range tests {
		t.Run(testName, func(t *testing.T) {
			db := sql.DB{}
			migrator, _ := New(&db, WithDialect(params.Dial), WithRootDirectory("./testdata/"))

			if migrator.dialect == nil {
				t.Fatalf("Migrator should not have a nil dialect")
			}
		})
	}
}
