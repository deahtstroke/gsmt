package integration

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

type ColumnResult struct {
	ColumnName string
	DataType   string
}

func InitialSetup(ctx context.Context) (*postgres.PostgresContainer, *sql.DB, error) {
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

	if err != nil {
		return nil, nil, err
	}

	connectionUrl, err := postgresC.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, nil, err
	}

	db, err := sql.Open("postgres", connectionUrl)
	if err != nil {
		return nil, nil, err
	}

	return postgresC, db, nil
}

func AllTablesExist(db *sql.DB, tables ...string) (bool, error) {
	if len(tables) == 0 {
		return false, fmt.Errorf("Tables parameter is empty")
	}

	placeholders := make([]string, len(tables))
	args := make([]interface{}, len(tables))

	for i, table := range tables {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = table
	}

	query := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM information_schema.tables
		WHERE table_schema = 'public'
			AND table_name IN (%s)`,
		strings.Join(placeholders, ","))

	var count int
	err := db.QueryRow(query, args...).Scan(&count)

	if err != nil {
		log.Printf("Error querying tables: %v", err)
		return false, fmt.Errorf("Error querying tables: %v", err)
	}

	return count == len(tables), nil
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

func GsmtSchemaExists(db *sql.DB) (bool, error) {
	rows, err := db.Query(`
	SELECT column_name, data_type
	FROM information_schema.columns
	WHERE table_name = 'gsmt_migrations'
	`)

	if err != nil {
		return false, fmt.Errorf("Error querying schema: %v", err)
	}

	defer rows.Close()

	columns := []ColumnResult{}
	for rows.Next() {
		var c ColumnResult
		if err := rows.Scan(&c.ColumnName, &c.DataType); err != nil {
			return false, fmt.Errorf("Error scanning columns and data types: %v", err)
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

	return containsAll(columns, expectedColumns), nil
}
