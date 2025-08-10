package store

import (
	"context"

	"github.com/deahtstroke/gsmt/pkg/schema"
)

type MetadataStore interface {
	SetupMetadataTables(ctx context.Context) error
	GetAppliedChecksums(ctx context.Context, table string) (map[string]string, error)
	RecordSchemaScript(ctx context.Context, script schema.MigrationScript) error
}
