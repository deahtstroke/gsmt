# gsmt

Schema migration tool for databases

GSMT is a lightweight embedded-schema and data migration library for Go.
It helps you apply, track and validate schema and data changes in your database
at runtime.

## Features

- Tracks applied migration in a metadata table
- Supports both schema and data migrations
- Works with both embedded and local filesystem `fs.FS`

## Installation

```bash
go get github.com/deahtstroke/gsmt
```

Or add it to your module:

```bash
go get github.com/deahtstroke/gsmt@latest
```

## Usage

### Basic Example Using Embedded Filesystem

Given a file system that looks like this

``` pgsql
main.go
migrations/
├── schema/
│   ├── 001_create_users_table.sql
│   └── 002_add_index_to_users.sql
└── data/
    ├── 001_seed_users.sql
    └── 002_update_roles.sql
go.mod
go.sum
```

```go
package main

import (
    "embed"
    "log"

    "github.com/deahtstroke/gsmt/pkg/dialect"
    "github.com/deahtstroke/gsmt/pkg/migrator"
    "github.com/deahtstroke/gsmt/pkg/store"
)

//go:embed migrations/schema
var schemaFS embed.FS

//go:embed mirgations/data
var dataFS embed.FS

func main() {
    db, _ := sql.Open("postgres", "postgres://user:pass@localhost/dbname?sslmode=disable")
    migrator, err := migrator.NewMigrator(migrator.MigratorOpts{
        Schema: schemaFS,
        Data: dataFS,
        Store := store.NewSQLStore(db, dialect.Postgres())
    })

    if err != nil {
        log.Fatal(err)
    }

    if err := migrator.ApplyMigrations(context.Background()); err != nil {
        log.Fatal(err)
    }

    log.Println("Migrations applied successfully")
}

```

## Migrator Configuration

| Option | Description | Required |
| ------ | ----------- | :--------: |
| Store  | Metadata store for applied migrations, changes based on SQL dialect | ✅ |
| Schema | `fs.FS` filesystem that contains the schema migrations | ✅ |
| Data   | `fs.FS` filesystem that contains data migrations | ❌ |
