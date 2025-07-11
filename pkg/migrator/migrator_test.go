package migrator

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/deahtstroke/gsmt/pkg/dialect"

	"github.com/DATA-DOG/go-sqlmock"
)

func GetTestDataFs() fs.FS {
	cwd, err := os.Getwd()
	if err != nil {
		return nil
	}
	return os.DirFS(path.Join(cwd, "testdata"))
}

func Test_new_migrator_should_have_all_scripts(t *testing.T) {
	db := sql.DB{}

	testDirectory := GetTestDataFs()
	fileCount, err := getScriptCount(testDirectory)
	if err != nil {
		t.Fatalf("Error getting script count: %v", err)
	}
	migrator, err := New(&db, WithSchema(testDirectory), WithDialect(dialect.Postgres()))

	if err != nil {
		t.Fatalf("Not expecting error, got: %v", err)
	}
	if len(migrator.schemaScripts) != fileCount {
		t.Fatalf("Wrong amount of test scripts. Found %d scripts", len(migrator.schemaScripts))
	}
}

func Test_new_migrator_without_dialect_should_panic(t *testing.T) {
	db := sql.DB{}

	_, err := New(&db, WithSchema(GetTestDataFs()))
	if err == nil {
		t.Fatalf("Expecting error, found none")
	}

	errMessage := "No dialect selected"
	if err.Error() != errMessage {
		t.Fatalf("Error message is not correct. Expected: %s, Actual: %s", errMessage, err.Error())
	}
}

func Test_fetchAppliedChecksums(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("An error %s was not expected when opening stub db connection", err)
	}

	defer db.Close()

	rows := sqlmock.NewRows([]string{"script_name", "checksum"}).
		AddRow("test1.sql", "1").
		AddRow("test2.sql", "2")
	mock.ExpectQuery("SELECT script_name, checksum FROM gsmt_migrations").
		WillReturnRows(rows)

	migrator := Migrator{
		Db: db,
	}

	result, err := migrator.fetchAppliedChecksums(context.Background())
	if err != nil {
		t.Fatalf("Not expecting error, found: %v", err)
	}

	if test1, ok := result["test1.sql"]; !ok || test1 != "1" {
		t.Fatalf("Unable to find test1.sql or test1.sql checksum is not correct")
	}

	if test2, ok := result["test2.sql"]; !ok || test2 != "2" {
		t.Fatal("Unable to find test2.sql or test2.sql checksum is not correct")
	}

	assertMockExpectations(t, mock)
}

func Test_fetchAppliedChecksums_sqlerror(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("An error %s was not expected when opening stub db connection", err)
	}

	defer db.Close()

	mock.ExpectQuery("SELECT script_name, checksum FROM gsmt_migrations").
		WillReturnError(fmt.Errorf("Error while retrieving rows"))

	migrator := Migrator{
		Db: db,
	}

	_, err = migrator.fetchAppliedChecksums(context.Background())
	if err == nil {
		t.Fatalf("Not expecting error, found: %v", err)
	}

	assertMockExpectations(t, mock)
}

func Test_ApplyMigrations_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("An error %v was not expected when opening stub db connection", err)
	}

	defer db.Close()

	migrator, err := New(db, WithSchema(GetTestDataFs()), WithDialect(dialect.Postgres()))

	if err != nil {
		t.Fatalf("Error creating migrator: %v", err)
	}
	exists := sqlmock.NewRows([]string{"exists"}).AddRow("true")
	emptyChecksum := sqlmock.NewRows([]string{"script_name", "checksum"})

	mock.ExpectQuery(`SELECT EXISTS`).WillReturnRows(exists)
	mock.ExpectQuery("SELECT script_name, checksum FROM gsmt_migrations").
		WillReturnRows(emptyChecksum)

	addMigratorInteractions(mock, migrator)

	err = migrator.ApplyMigrations()

	if err != nil {
		t.Fatalf("Not expecting error, found: %v", err)
	}

	assertMockExpectations(t, mock)
}

func Test_ApplyMigrations_AllChecksumsApplied(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("An error %v was not expected when opening stub db connection", err)
	}

	defer db.Close()

	migrator, err := New(db, WithSchema(GetTestDataFs()), WithDialect(dialect.Postgres()))

	exists := sqlmock.NewRows([]string{"exists"}).AddRow("true")
	checksums := sqlmock.NewRows([]string{"script_name", "checksum"}).
		AddRow("test1.sql", encode(migrator.schemaScripts[0].Content)).
		AddRow("test2.sql", encode(migrator.schemaScripts[1].Content))

	mock.ExpectQuery(`SELECT EXISTS`).WillReturnRows(exists)
	mock.ExpectQuery(`SELECT script_name, checksum FROM gsmt_migrations`).
		WillReturnRows(checksums)

	err = migrator.ApplyMigrations()

	if err != nil {
		t.Fatalf("Not expecting error, found: %v", err)
	}

	assertMockExpectations(t, mock)
}

func Test_ApplyMigrations_DifferentChecksumError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("An error %v was not expected when opening stub db connection", err)
	}

	defer db.Close()

	migrator, err := New(db, WithSchema(GetTestDataFs()), WithDialect(dialect.Postgres()))

	exists := sqlmock.NewRows([]string{"exists"}).AddRow("true")
	checksums := sqlmock.NewRows([]string{"script_name", "checksum"}).
		AddRow("test1.sql", encode(migrator.schemaScripts[0].Content)).
		AddRow("test2.sql", "1")

	mock.ExpectQuery(`SELECT EXISTS`).WillReturnRows(exists)
	mock.ExpectQuery(`SELECT script_name, checksum FROM gsmt_migrations`).
		WillReturnRows(checksums)

	err = migrator.ApplyMigrations()

	if err == nil {
		t.Fatalf("Expecting error, found none")
	}

	assertMockExpectations(t, mock)
}

func Test_ApplyMigrations_RollbackOnScriptExecuteError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("An error %v was not expected when opening stub db connection", err)
	}

	defer db.Close()

	migrator, err := New(db, WithSchema(GetTestDataFs()), WithDialect(dialect.Postgres()))
	exists := sqlmock.NewRows([]string{"exists"}).AddRow("true")
	checksums := sqlmock.NewRows([]string{"script_name", "checksum"})

	mock.ExpectQuery(`SELECT EXISTS`).WillReturnRows(exists)
	mock.ExpectQuery(`SELECT script_name, checksum FROM gsmt_migrations`).
		WillReturnRows(checksums)
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(migrator.schemaScripts[0].Content)).WillReturnError(fmt.Errorf("Random error!"))
	mock.ExpectRollback()

	err = migrator.ApplyMigrations()

	if err == nil {
		t.Fatalf("Expecting error, found none")
	}

	assertMockExpectations(t, mock)
}

func Test_ApplyMigrations_RollbackOnMigrationRowInsertError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("An error %v was not expected when opening stub db connection", err)
	}

	defer db.Close()

	migrator, err := New(db, WithSchema(GetTestDataFs()), WithDialect(dialect.Postgres()))

	exists := sqlmock.NewRows([]string{"exists"}).AddRow("true")
	checksums := sqlmock.NewRows([]string{"script_name", "checksum"})

	mock.ExpectQuery(`SELECT EXISTS`).WillReturnRows(exists)
	mock.ExpectQuery(`SELECT script_name, checksum FROM gsmt_migrations`).
		WillReturnRows(checksums)
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(migrator.schemaScripts[0].Content)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO gsmt_migrations (checksum, execution_time_ms, script_name, script_content)`)).
		WillReturnError(fmt.Errorf("Error inserting"))
	mock.ExpectRollback()

	err = migrator.ApplyMigrations()

	if err == nil {
		t.Fatalf("Expecting error, found none")
	}

	assertMockExpectations(t, mock)
}

func assertMockExpectations(t *testing.T, mock sqlmock.Sqlmock) {
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("Mock expectations were not met: %v", err)
	}
}

func getScriptCount(directory fs.FS) (int, error) {
	count := 0
	fs.WalkDir(directory, ".", func(path string, d fs.DirEntry, err error) error {
		if filepath.Ext(d.Name()) == ".sql" {
			count++
		}
		return nil
	})

	return count, nil
}

func addMigratorInteractions(mock sqlmock.Sqlmock, migrator *Migrator) {
	for _, script := range migrator.schemaScripts {
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(script.Content)).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO gsmt_migrations (checksum, execution_time_ms, script_name, script_content)`)).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()
	}
}
