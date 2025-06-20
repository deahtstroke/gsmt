package integration

import (
	"context"
	"database/sql"
	"github.com/deahtstroke/gsmt/pkg/dialect"
	"github.com/deahtstroke/gsmt/pkg/migrate"
	"log"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	_ "github.com/lib/pq"
)

type ColumnResult struct {
	ColumnName string
	DataType   string
}

func Test_PostgresDialectMigrationsShouldBeSuccessful(t *testing.T) {
	ctx := context.Background()
	dbName := "test"
	dbUser := "user"
	dbPassword := "password"
	dbImageVersion := "postgres:17"

	postgresC, err := postgres.Run(ctx,
		dbImageVersion,
		postgres.WithDatabase(dbName),
		postgres.WithUsername(dbUser),
		postgres.WithPassword(dbPassword),
		postgres.BasicWaitStrategies())

	defer func() {
		if err := testcontainers.TerminateContainer(postgresC); err != nil {
			log.Printf("Failed to terminate container: %s", err)
		}
	}()

	if err != nil {
		t.Errorf("Failed to start container: %s", err)
	}

	connectionUrl, err := postgresC.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Errorf("Failed to get connection string from postgres container: %v", err)
	}

	db, err := sql.Open("postgres", connectionUrl)
	if err != nil {
		t.Errorf("Error opening DB connection to test database: %v", err)
	}

	migrator, err := migrate.New(db, migrate.WithRootDirectory("./migrations/scripts"), migrate.WithDialect(dialect.NewPostgresDialect()))
	if err != nil {
		t.Fatalf("Error creating migrator: %v", err)
	}

	err = migrator.ApplyMigrations()

	if err != nil {
		t.Errorf("Error applying migrations: %v", err)
	}

	rows, err := db.Query(`
	SELECT column_name, data_type
	FROM information_schema.columns
	WHERE table_name = 'gsmt_migrations'
	`)

	if err != nil {
		t.Errorf("Error getting column data: %v", err)
	}

	defer rows.Close()

	columns := []ColumnResult{}
	for rows.Next() {
		var c ColumnResult
		if err := rows.Scan(&c.ColumnName, &c.DataType); err != nil {
			t.Errorf("%v", err)
		}
		columns = append(columns, c)
	}

	log.Printf("%v", columns)

	expectedColumns := []ColumnResult{
		{ColumnName: "id", DataType: "integer"},
		{ColumnName: "checksum", DataType: "text"},
		{ColumnName: "applied_at", DataType: "timestamp with time zone"},
		{ColumnName: "execution_time_ms", DataType: "bigint"},
		{ColumnName: "script_name", DataType: "text"},
		{ColumnName: "script_content", DataType: "text"},
	}

	if !containsAll(columns, expectedColumns) {
		t.Error("Missing columns")
	}
}

func contains(slice []ColumnResult, cr ColumnResult) bool {
	for _, column := range slice {
		if cr.ColumnName == column.ColumnName && cr.DataType == column.DataType {
			return true
		}
	}
	log.Printf("Column %v not found", cr.ColumnName)
	return false
}

func containsAll(slice1 []ColumnResult, slice2 []ColumnResult) bool {
	for _, cr := range slice1 {
		if !contains(slice2, cr) {
			return false
		}
	}
	return true
}
