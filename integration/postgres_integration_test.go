package integration

import (
	"context"
	"embed"
	"log"
	"os"
	"testing"

	"github.com/deahtstroke/gsmt/pkg/dialect"
	"github.com/deahtstroke/gsmt/pkg/migrator"
	"github.com/deahtstroke/gsmt/pkg/store"

	"github.com/testcontainers/testcontainers-go"

	_ "github.com/lib/pq"
)

//go:embed migrations/scripts/*.sql
var schemaFs embed.FS

//go:embed migrations/data/*.sql
var dataFs embed.FS

func Test_PostgresDialectMigrationsShouldBeSuccessful_NoEmbeddedFS(t *testing.T) {
	ctx := context.Background()
	postgresC, db, err := InitialSetup(ctx)

	if err != nil {
		t.Errorf("Failed setup: %s", err)
	}

	defer func() {
		if err := testcontainers.TerminateContainer(postgresC); err != nil {
			log.Printf("Failed to terminate container: %s", err)
		}
	}()
	fs := os.DirFS("./migrations/scripts/")
	migrator, err := migrator.NewMigrator(migrator.MigratorOpts{
		Schema: fs,
		Store:  store.NewSQLStore(db, dialect.Postgres()),
	})
	if err != nil {
		t.Fatalf("Error creating migrator: %v", err)
	}

	err = migrator.ApplyMigrations(context.Background())

	if err != nil {
		t.Errorf("Error applying migrations: %v", err)
	}

	ok, err := GsmtSchemaExists(db)
	if err != nil {
		t.Error(err)
	}

	if !ok {
		t.Error("Gsmt schema does not exist")
	}

	ok, err = AllTablesExist(db, "department", "employee")
	if err != nil {
		t.Error(err)
	}

	if !ok {
		t.Error("Some tables are missing")
	}
}

func Test_PostgresDialectMigrationsShouldBeSuccessful_EmbeddedFs(t *testing.T) {
	ctx := context.Background()

	postgresC, db, err := InitialSetup(ctx)
	if err != nil {
		t.Errorf("Setup failed: %s", err)
	}

	defer func() {
		if err := testcontainers.TerminateContainer(postgresC); err != nil {
			log.Printf("Failed to terminate container: %s", err)
		}
	}()

	migrator, err := migrator.NewMigrator(migrator.MigratorOpts{
		Schema: schemaFs,
		Store:  store.NewSQLStore(db, dialect.Postgres()),
	})
	if err != nil {
		t.Fatalf("Error creating migrator: %v", err)
	}

	err = migrator.ApplyMigrations(context.Background())

	if err != nil {
		t.Errorf("Error applying migrations: %v", err)
	}

	ok, err := GsmtSchemaExists(db)
	if err != nil {
		t.Error(err)
	}

	if !ok {
		t.Error("Gsmt schema does not exist")
	}
}

func Test_PostgresDialect_MigationsWithDataShouldBeSuccessul(t *testing.T) {
	ctx := context.Background()

	postgresC, db, err := InitialSetup(ctx)
	if err != nil {
		t.Errorf("Setup failed: %s", err)
	}

	defer func() {
		if err := testcontainers.TerminateContainer(postgresC); err != nil {
			log.Printf("Failed to terminate container: %s", err)
		}
	}()

	migrator, err := migrator.NewMigrator(migrator.MigratorOpts{
		Schema: schemaFs,
		Data:   dataFs,
		Store:  store.NewSQLStore(db, dialect.Postgres()),
	})
	if err != nil {
		t.Errorf("Error creating migrator: %s", err)
	}

	err = migrator.ApplyMigrations(context.Background())
	if err != nil {
		t.Errorf("Error applying migrations: %s", err)
	}

	ok, err := GsmtSchemaExists(db)
	if err != nil {
		t.Error(err)
	}

	if !ok {
		t.Errorf("GSMT schema does not exist")
	}

	var departmentCount int
	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM department
		`).Scan(&departmentCount)
	if err != nil {
		t.Error(err)
	}
	if departmentCount != 2 {
		t.Error("Wrong department count")
	}

	var employeeCount int
	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM employee
		`).Scan(&employeeCount)
	if err != nil {
		t.Error(err)
	}

	if employeeCount != 4 {
		t.Error("Employee count is less than the actual amount")
	}
}
