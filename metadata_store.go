package gsmt

import (
	"context"
)

type Store interface {
	SetupMetadataTables(ctx context.Context) error
	GetAppliedChecksums(ctx context.Context, table TableName) (map[string]string, error)
	RecordSchemaScript(ctx context.Context, script MigrationScript) error
	RecordDataScript(ctx context.Context, script MigrationScript) error
}
