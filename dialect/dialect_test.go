package dialect

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	_ "modernc.org/sqlite"
)

type DialectTestSuite struct {
	Dialect Dialect
	DB      *sql.DB
}

func TestPostgresDialect(t *testing.T) {
	ctx := context.Background()
	db, container, err := createPostgresContainer(ctx, t)
	require.NoError(t, err)

	t.Cleanup(func() {
		db.Close()
		if err := container.Terminate(ctx); err != nil {
			t.Fatalf("Failed to terminate postgres container: %s", err)
		}
	})

	RunDialectTests(t, DialectTestSuite{Dialect: NewPostgres(), DB: db})
}

func TestSqlite3Dialect(t *testing.T) {
	db, file, err := connectToSqlite(t)
	require.NoError(t, err)

	t.Cleanup(func() {
		db.Close()
		os.Remove(file.Name())
	})

	RunDialectTests(t, DialectTestSuite{Dialect: NewSqlite3(), DB: db})
}

func connectToSqlite(t *testing.T) (*sql.DB, *os.File, error) {
	tempFile, err := os.CreateTemp("", "testdb-*.db")
	require.NoError(t, err)

	path := tempFile.Name()
	db, err := sql.Open("sqlite", path)
	require.NoError(t, err)

	return db, tempFile, nil
}

func RunDialectTests(t *testing.T, suite DialectTestSuite) {
	table := "gsmt_metadata"

	_, err := suite.DB.Exec(suite.Dialect.CreateMetadataTable(table))
	require.NoError(t, err)

	var exists bool
	err = suite.DB.QueryRow(suite.Dialect.TableExists(), table).
		Scan(&exists)
	require.NoError(t, err)
	require.True(t, exists)

	_, err = suite.DB.Exec(
		suite.Dialect.InsertMetadata(table),
		"abc",
		int64(10),
		"001_init.sql",
		"CREATE TABLE X",
	)
	require.NoError(t, err)

	rows, err := suite.DB.Query(suite.Dialect.GetChecksums(table))
	require.NoError(t, err)

	var (
		scriptName string
		checksum   string
	)
	rows.Scan(&scriptName, &checksum)
	rows.Close()
}

func createPostgresContainer(ctx context.Context, t *testing.T) (*sql.DB, *postgres.PostgresContainer, error) {
	container, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithUsername("user"),
		postgres.WithPassword("password"),
		postgres.WithDatabase("testdb"))
	require.NoError(t, err)

	// Wait 10 seconds for PG to start
	time.Sleep(10 * time.Second)

	connString, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	db, err := sql.Open("postgres", connString)
	return db, container, err
}
