package dialect

import (
	"database/sql"
)

type Dialect interface {
	Placeholder(i int) string
	SetupMetadataTables(db *sql.DB) error
}
