package migrator

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/deahtstroke/duck-assert/mock"
	"github.com/deahtstroke/gsmt/pkg/schema"
	"github.com/deahtstroke/gsmt/pkg/utils"
)

type MockDialect struct {
	getMetadataTablesCalled   bool
	setupMetadataTablesCalled bool
	placeHolderCalled         bool
	errToReturn               error
}

type MockMetadataStore struct {
	mock.Mock
}

func (m *MockMetadataStore) SetupMetadataTables(ctx context.Context) error {
	ret := m.Called("SetupMetadataTables", ctx)
	return ret.Error(0)
}

func (m *MockMetadataStore) GetAppliedChecksums(ctx context.Context, table string) (map[string]string, error) {
	ret := m.Called("GetAppliedChecksums", ctx, table)
	return ret.Get(0).(map[string]string), ret.Error(1)
}

func (m *MockMetadataStore) RecordSchemaScript(ctx context.Context, script schema.MigrationScript) error {
	ret := m.Called("RecordSchemaScript", ctx, script)
	return ret.Error(0)
}

func Test_ensureMetadataTables_ShouldBeSuccessful(t *testing.T) {
	schemaFS := os.DirFS("./testdata/schema/")
	dataFS := os.DirFS("./testdata/data/")
	mockStore := new(MockMetadataStore)
	ctx := context.Background()
	mockStore.On("SetupMetadataTables", ctx).ThenReturn(error(nil))
	m, err := NewMigrator(MigratorOpts{
		schemaFS: schemaFS,
		dataFS:   dataFS,
		store:    mockStore,
	})

	if err != nil {
		t.Errorf("Error creating migrator")
	}

	err = m.ensureMetadataTables(ctx)
	if err != nil {
		t.Errorf("Not expecting an error, got: %v", err)
	}

	mockStore.AssertCalled(t, "SetupMetadataTables", ctx)
	mockStore.AssertNumberOfCalls(t, "SetupMetadataTables", 1)
}

func Test_ensureMetadataTables_FailOnErrorReturned(t *testing.T) {
	schemaFS := os.DirFS("./testdata/schema/")
	dataFS := os.DirFS("./testdata/data/")
	mockStore := new(MockMetadataStore)
	ctx := context.Background()
	m, err := NewMigrator(MigratorOpts{
		schemaFS: schemaFS,
		dataFS:   dataFS,
		store:    mockStore,
	})

	if err != nil {
		t.Errorf("Error creating migrator")
	}
	mockStore.On("SetupMetadataTables", ctx).ThenReturn(fmt.Errorf("Error while setting up tables"))

	err = m.ensureMetadataTables(ctx)
	if err == nil {
		t.Errorf("Expecting error, found none")
	}

	mockStore.AssertCalled(t, "SetupMetadataTables", ctx)
	mockStore.AssertNumberOfCalls(t, "SetupMetadataTables", 1)
}

func Test_migrateSchema_ShouldBeSuccessful(t *testing.T) {
	schemaFS := os.DirFS("./testdata/schema")
	mockStore := new(MockMetadataStore)
	ctx := context.Background()

	hashes := getScriptHashes(schemaFS)

	for k := range hashes {
		mockStore.On("RecordSchemaScript", ctx, mock.MatchedBy(func(s schema.MigrationScript) bool {
			return s.Hash == hashes[k]
		})).ThenReturn(nil)
	}

	mockStore.On("GetAppliedChecksums", ctx, schema.SchemaMigrationsTable).
		ThenReturn(map[string]string{}, nil)

	m, _ := NewMigrator(MigratorOpts{
		schemaFS: schemaFS,
		store:    mockStore,
	})

	err := m.migrateSchema(ctx)
	if err != nil {
		t.Errorf("%s", err)
	}

	mockStore.AssertCalled(t, "GetAppliedChecksums", ctx, schema.SchemaMigrationsTable)
	mockStore.AssertNumberOfCalls(t, "GetAppliedChecksums", 1)
	mockStore.AssertNumberOfCalls(t, "RecordSchemaScript", len(hashes))
}

func Test_mirgateSchema_SuccessOnAllHashesApplied(t *testing.T) {
	schemaFS := os.DirFS("./testdata/schema")
	mockStore := new(MockMetadataStore)
	ctx := context.Background()

	hashes := getScriptHashes(schemaFS)

	mockStore.On("GetAppliedChecksums", ctx, schema.SchemaMigrationsTable).
		ThenReturn(hashes, nil)

	m, _ := NewMigrator(MigratorOpts{
		schemaFS: schemaFS,
		store:    mockStore,
	})

	err := m.migrateSchema(ctx)
	if err != nil {
		t.Errorf("%s", err)
	}

	mockStore.AssertCalled(t, "GetAppliedChecksums", ctx, schema.SchemaMigrationsTable)
	mockStore.AssertNumberOfCalls(t, "GetAppliedChecksums", 1)
	mockStore.AssertNotCalled(t, "RecordSchemaScript")
}

func Test_migrateSchema_SuccessOnPartialRecordings(t *testing.T) {
	schemaFS := os.DirFS("./testdata/schema")
	mockStore := new(MockMetadataStore)
	ctx := context.Background()
	hashes := getScriptHashes(schemaFS)

	count := 0
	for h := range hashes {
		if count%2 == 0 {
			hashes[h] = ""
		}
		count++
	}

	mockStore.On("GetAppliedChecksums", ctx, schema.SchemaMigrationsTable).
		ThenReturn(map[string]string{}, fmt.Errorf("Error fetching applied checksums"))

	count = 0
	for h := range hashes {
		if count%2 != 0 {
			mockStore.On("RecordSchemaScript", mock.MatchedBy(func(s schema.MigrationScript) bool {
				return s.Hash == hashes[h]
			}))
		}
		count++
	}

	m, _ := NewMigrator(MigratorOpts{
		schemaFS: schemaFS,
		store:    mockStore,
	})

	err := m.migrateSchema(ctx)
	if err == nil {
		t.Errorf("Expecting error, found none")
	}
}

func Test_migrateSchema_ErrorWhileFetchingAppliedChecksums(t *testing.T) {
	schemaFS := os.DirFS("./testdata/schema")
	mockStore := new(MockMetadataStore)
	ctx := context.Background()

	mockStore.On("GetAppliedChecksums", ctx, schema.SchemaMigrationsTable).
		ThenReturn(map[string]string{}, fmt.Errorf("Error fetching applied checksums"))

	m, _ := NewMigrator(MigratorOpts{
		schemaFS: schemaFS,
		store:    mockStore,
	})

	err := m.migrateSchema(ctx)
	if err == nil {
		t.Errorf("Expecting error, found none")
	}

	mockStore.AssertCalled(t, "GetAppliedChecksums", ctx, schema.SchemaMigrationsTable)
	mockStore.AssertNumberOfCalls(t, "GetAppliedChecksums", 1)
	mockStore.AssertNotCalled(t, "RecordSchemaScript")
}

func Test_migrateSchema_ErrorWhileRecordingSchemaChanges(t *testing.T) {
	schemaFS := os.DirFS("./testdata/schema")
	mockStore := new(MockMetadataStore)
	ctx := context.Background()

	mockStore.On("GetAppliedChecksums", ctx, schema.SchemaMigrationsTable).
		ThenReturn(map[string]string{}, nil)

	hashes := getScriptHashes(schemaFS)

	count := 0
	for k := range hashes {
		if count%2 == 0 {
			mockStore.On("RecordSchemaScript", ctx, mock.MatchedBy(func(s schema.MigrationScript) bool {
				return s.Hash == hashes[k]
			})).ThenReturn(fmt.Errorf("Error recording script with hash %s", hashes[k]))
		}
		mockStore.On("RecordSchemaScript", ctx, mock.MatchedBy(func(s schema.MigrationScript) bool {
			return s.Hash == hashes[k]
		})).ThenReturn(nil)

		count++
	}

	m, _ := NewMigrator(MigratorOpts{
		schemaFS: schemaFS,
		store:    mockStore,
	})

	err := m.migrateSchema(ctx)

	if err == nil {
		t.Errorf("Expecting error, found none")
	}

	mockStore.AssertCalled(t, "GetAppliedChecksums", ctx, schema.SchemaMigrationsTable)
	mockStore.AssertNumberOfCalls(t, "GetAppliedChecksums", 1)
	mockStore.AssertNumberOfCalls(t, "RecordSchemaScript", len(hashes)-1)
}

func getScriptHashes(dir fs.FS) map[string]string {
	hashes := map[string]string{}
	err := fs.WalkDir(dir, ".", func(path string, d fs.DirEntry, err error) error {
		if filepath.Ext(path) == ".sql" {
			content, err := utils.ReadFileContent(dir, path)
			if err != nil {
				return err
			}

			hashes[d.Name()] = utils.Encode(content)
		}
		return nil
	})

	if err != nil {
		return nil
	}

	return hashes
}
