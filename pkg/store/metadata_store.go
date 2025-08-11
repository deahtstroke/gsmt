package store

import (
	"context"

	"github.com/deahtstroke/gsmt/pkg/data"
)

type MetadataStore interface {
	SetupMetadataTables(ctx context.Context) error
	GetAppliedChecksums(ctx context.Context, table string) (map[string]string, error)
	RecordSchemaScript(ctx context.Context, script data.MigrationScript) error
	RecordDataScript(ctx context.Context, script data.MigrationScript) error
}
