package gsmt

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func Test_CreateTableIfNotExistsShouldBeSuccessful(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Errorf("Error creating sqlmock: %v", err)
	}

	defer db.Close()

	rows := sqlmock.NewRows([]string{"exists"}).
		AddRow("false")
	mock.ExpectQuery("SELECT EXISTS").
		WillReturnRows(rows)

	mock.ExpectExec("CREATE TABLE IF NOT EXISTS").
		WillReturnResult(sqlmock.NewResult(1, 1))

	ddl := `
	CREATE TABLE IF NOT EXISTS some_table (
	id serial PRIMARY KEY,
	name TEXT NOT NULL
	);
	`
	ctx := context.Background()
	store := NewSQLStore(db, Postgres())
	err = store.createTableIfNotExists(ctx, "some_table", ddl)

	if err != nil {
		t.Errorf("Not expecting error, got: %v", err)
	}

	if err = mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Some expectations were not met: %v", err)
	}
}

func Test_CreateTableIfNotExists_ErrorWhenDDLIsEmpty(t *testing.T) {
	input := []struct {
		description string
		ddl         string
	}{
		{
			description: "All spaces",
			ddl:         "   ",
		},
		{
			description: "Empty string",
			ddl:         "",
		},
	}

	for _, test := range input {
		t.Run(test.description, func(t *testing.T) {
			db, _, err := sqlmock.New()
			if err != nil {
				t.Errorf("Error creating sqlmock: %v", err)
			}

			defer db.Close()
			store := NewSQLStore(db, Postgres())
			ctx := context.Background()
			err = store.createTableIfNotExists(ctx, "some_table", test.ddl)

			if err == nil {
				t.Error("Expecting error, found none")
			}
		})
	}
}

func Test_CreateTableIfNotExists_ShouldNotExecuteDDLIfTableExists(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Errorf("Error creating sqlmock: %v", err)
	}

	defer db.Close()

	rows := sqlmock.NewRows([]string{"exists"}).
		AddRow("true")
	mock.ExpectQuery("SELECT EXISTS").
		WillReturnRows(rows)

	ddl := `
	CREATE TABLE IF NOT EXISTS some_table (
	id serial PRIMARY KEY,
	name TEXT NOT NULL
	);
	`

	store := NewSQLStore(db, Postgres())
	ctx := context.Background()
	err = store.createTableIfNotExists(ctx, "some_table", ddl)

	if err != nil {
		t.Errorf("Not expecting error, got: %v", err)
	}

	if err = mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Some expectations were not met: %v", err)
	}
}

func Test_CreateTableIfNotExists_ErrorWhenQueryingExists(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Errorf("Error creating sqlmock: %v", err)
	}

	defer db.Close()

	mock.ExpectQuery("SELECT EXISTS").
		WillReturnError(errors.New("Error while querying db"))

	ddl := `
	CREATE TABLE IF NOT EXISTS some_table (
	id serial PRIMARY KEY,
	name TEXT NOT NULL
	);
	`

	store := NewSQLStore(db, Postgres())
	ctx := context.Background()
	err = store.createTableIfNotExists(ctx, "some_table", ddl)

	if err == nil {
		t.Error("Expecting error, found none")
	}

	if err = mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Mock expectations were not met: %v", err)
	}
}

func Test_CreateTableIfNotExists_ErrorWhenExecutingDDL(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Errorf("Error creating sqlmock: %v", err)
	}

	defer db.Close()

	rows := sqlmock.NewRows([]string{"exists"}).
		AddRow("false")
	mock.ExpectQuery("SELECT EXISTS").
		WillReturnRows(rows)
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS").
		WillReturnError(errors.New("Error executing DDL"))

	ddl := `
	CREATE TABLE IF NOT EXISTS some_table (
	id serial PRIMARY KEY,
	name TEXT NOT NULL
	),
	`

	store := NewSQLStore(db, Postgres())
	ctx := context.Background()
	err = store.createTableIfNotExists(ctx, "some_table", ddl)

	if err == nil {
		t.Error("Expecting error, found none")
	}

	if err = mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Mock expectations were not met: %v", err)
	}
}

func Test_GetAppliedChecksums_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Errorf("Error creating sqlmock: %v", err)
	}

	defer db.Close()

	rows := sqlmock.NewRows([]string{"script_name", "checksum"}).
		AddRow("v1__init_schema.sql", "123897489123891").
		AddRow("v2__add_employee.sql", "21837189231238")

	mock.ExpectQuery("SELECT script_name, checksum").WillReturnRows(rows)

	store := NewSQLStore(db, Postgres())
	ctx := context.Background()

	result, err := store.GetAppliedChecksums(ctx, SchemaMigrationsTable)
	if err != nil {
		t.Errorf("Not expecting error, found: %v", err)
	}

	if result["v1__init_schema.sql"] != "123897489123891" {
		t.Errorf("Error fetching first script")
	}

	if result["v2__add_employee.sql"] != "21837189231238" {
		t.Error("Error fetching second script")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Mock expectations were not met: %v", err)
	}
}

func Test_GetAppliedChecksums_SucessEmptyMap(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Errorf("Error creating sqlmock: %v", err)
	}

	defer db.Close()

	rows := sqlmock.NewRows([]string{"script_name", "checksum"})
	mock.ExpectQuery("SELECT script_name, checksum").WillReturnRows(rows)

	store := NewSQLStore(db, Postgres())
	ctx := context.Background()

	result, err := store.GetAppliedChecksums(ctx, SchemaMigrationsTable)
	if err != nil {
		t.Errorf("Not expecting error, found: %s", err)
	}

	if len(result) > 0 {
		t.Error("Not expecting map to have more than 1 entry")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Mock expectations were not met: %v", err)
	}
}

func Test_GetAppliedChecksums_ErrorQueryingDatabase(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Errorf("Error creating sqlmock: %v", err)
	}

	defer db.Close()

	mock.ExpectQuery("SELECT script_name, checksum").WillReturnError(fmt.Errorf("Error querying checksums"))

	store := NewSQLStore(db, Postgres())
	ctx := context.Background()

	_, err = store.GetAppliedChecksums(ctx, SchemaMigrationsTable)
	if err == nil {
		t.Errorf("Expecting error, found none")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Mock expectations were not met: %v", err)
	}
}

func Test_RecordSchemaScript_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Errorf("Error creating sqlmock: %s", err)
	}

	defer db.Close()

	script := MigrationScript{
		Hash:    "123412",
		Content: "SELECT 1",
		Name:    "v1__hello.sql",
	}

	mock.ExpectBegin()
	mock.ExpectExec(script.Content).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(fmt.Sprintf("INSERT INTO %s", SchemaMigrationsTable)).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	store := NewSQLStore(db, Postgres())
	ctx := context.Background()

	err = store.RecordSchemaScript(ctx, script)
	if err != nil {
		t.Errorf("Not expecting error, found: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Mock expectations were not: %v", err)
	}
}

func Test_RecordSchemaScript_ErrorBeginningTransaction(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Errorf("Error creating sqlmock: %s", err)
	}

	defer db.Close()

	script := MigrationScript{
		Hash:    "123412",
		Content: "SELECT 1",
		Name:    "v1__hello.sql",
	}

	mock.ExpectBegin().WillReturnError(fmt.Errorf("Error beginning transcation"))

	store := NewSQLStore(db, Postgres())
	ctx := context.Background()

	err = store.RecordSchemaScript(ctx, script)
	if err == nil {
		t.Error("Expecting error, found none")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Mock expectations were not: %v", err)
	}
}

func Test_RecordSchemaScript_ErrorInsertingScript(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Errorf("Error creating sqlmock: %s", err)
	}

	defer db.Close()

	script := MigrationScript{
		Hash:    "123412",
		Content: "SELECT 1",
		Name:    "v1__hello.sql",
	}

	mock.ExpectBegin()
	mock.ExpectExec(script.Content).WillReturnError(fmt.Errorf("Error inserting content"))

	store := NewSQLStore(db, Postgres())
	ctx := context.Background()

	err = store.RecordSchemaScript(ctx, script)
	if err == nil {
		t.Error("Expecting error, found none")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Mock expectations were not: %v", err)
	}
}

func Test_RecordSchemaScript_ErrorInsertingToSchemaTable(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Errorf("Error creating sqlmock: %s", err)
	}

	defer db.Close()

	script := MigrationScript{
		Hash:    "123412",
		Content: "SELECT 1",
		Name:    "v1__hello.sql",
	}

	mock.ExpectBegin()
	mock.ExpectExec(script.Content).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(fmt.Sprintf("INSERT INTO %s", SchemaMigrationsTable)).WillReturnError(fmt.Errorf("Error inserting to schema table"))

	store := NewSQLStore(db, Postgres())
	ctx := context.Background()

	err = store.RecordSchemaScript(ctx, script)
	if err == nil {
		t.Error("Expecting error, found none")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Mock expectations were not: %v", err)
	}
}

func Test_RecordSchemaScript_ErrorCommitingTransaction(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Errorf("Error creating sqlmock: %s", err)
	}

	defer db.Close()

	script := MigrationScript{
		Hash:    "123412",
		Content: "SELECT 1",
		Name:    "v1__hello.sql",
	}

	mock.ExpectBegin()
	mock.ExpectExec(script.Content).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(fmt.Sprintf("INSERT INTO %s", SchemaMigrationsTable)).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(fmt.Errorf("Error commiting tx"))

	store := NewSQLStore(db, Postgres())
	ctx := context.Background()

	err = store.RecordSchemaScript(ctx, script)
	if err == nil {
		t.Error("Expecting error, found none")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Mock expectations were not met: %v", err)
	}
}

func Test_RecordDataScript_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Errorf("Error creating sqlmock: %s", err)
	}

	defer db.Close()

	script := MigrationScript{
		Hash:    "12308768912",
		Content: "SELECT 1",
		Name:    "data_employee.sql",
	}

	mock.ExpectBegin()
	mock.ExpectExec(script.Content).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(fmt.Sprintf("INSERT INTO %s", DataMigrationsTable)).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	store := NewSQLStore(db, Postgres())
	ctx := context.Background()

	err = store.RecordDataScript(ctx, script)
	if err != nil {
		t.Errorf("Not expecting error, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Mock expectations were not met: %v", err)
	}
}

func Test_RecordDataScript_ErrorBeginningTx(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Errorf("Error creating sqlmock: %s", err)
	}

	defer db.Close()

	script := MigrationScript{
		Hash:    "12308768912",
		Content: "SELECT 1",
		Name:    "data_employee.sql",
	}

	mock.ExpectBegin().WillReturnError(fmt.Errorf("Error begining transaction"))

	store := NewSQLStore(db, Postgres())
	ctx := context.Background()

	err = store.RecordDataScript(ctx, script)
	if err == nil {
		t.Error("Expecting error, found none")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Mock expectations were not met: %v", err)
	}
}

func Test_RecordDataScript_ErrorRecordingDataScript(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Errorf("Error creating sqlmock: %s", err)
	}

	defer db.Close()

	script := MigrationScript{
		Hash:    "12308768912",
		Content: "SELECT 1",
		Name:    "data_employee.sql",
	}

	mock.ExpectBegin()
	mock.ExpectExec(script.Content).WillReturnError(fmt.Errorf("Error inserting data"))

	store := NewSQLStore(db, Postgres())
	ctx := context.Background()

	err = store.RecordDataScript(ctx, script)
	if err == nil {
		t.Error("Expecting error, found none")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Mock expectations were not met: %v", err)
	}
}

func Test_RecordDataScript_ErrorInsertingToDataTable(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Errorf("Error creating sqlmock: %s", err)
	}

	defer db.Close()

	script := MigrationScript{
		Hash:    "12308768912",
		Content: "SELECT 1",
		Name:    "data_employee.sql",
	}

	mock.ExpectBegin()
	mock.ExpectExec(script.Content).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(fmt.Sprintf("INSERT INTO %s", DataMigrationsTable)).WillReturnError(fmt.Errorf("Error inserting into gsmt-data-migrations"))

	store := NewSQLStore(db, Postgres())
	ctx := context.Background()

	err = store.RecordDataScript(ctx, script)
	if err == nil {
		t.Error("Expecting error, found none")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Mock expectations were not met: %v", err)
	}
}

func Test_RecordDataScript_ErrorCommitingTransaction(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Errorf("Error creating sqlmock: %s", err)
	}

	defer db.Close()

	script := MigrationScript{
		Hash:    "12308768912",
		Content: "SELECT 1",
		Name:    "data_employee.sql",
	}

	mock.ExpectBegin()
	mock.ExpectExec(script.Content).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(fmt.Sprintf("INSERT INTO %s", DataMigrationsTable)).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(fmt.Errorf("Error commiting transaction"))

	store := NewSQLStore(db, Postgres())
	ctx := context.Background()

	err = store.RecordDataScript(ctx, script)
	if err == nil {
		t.Error("Expecting error, found none")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Mock expectations were not met: %v", err)
	}
}
