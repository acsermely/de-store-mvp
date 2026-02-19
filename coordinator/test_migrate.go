//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	migrationsPath := "/home/csermely/prog/de-store-mvp/coordinator/migrations"

	// Check if directory exists
	if _, err := os.Stat(migrationsPath); os.IsNotExist(err) {
		fmt.Printf("ERROR: migrations directory does not exist: %s\n", migrationsPath)
		os.Exit(1)
	}

	// List files
	files, err := os.ReadDir(migrationsPath)
	if err != nil {
		fmt.Printf("ERROR: could not read migrations directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Migration files found:")
	for _, f := range files {
		fmt.Printf("  - %s\n", f.Name())
	}

	databaseURL := "postgres://postgres:postgres@localhost:5432/coordinator?sslmode=disable"

	// Use file:// URL scheme
	migrationURL := "file://" + migrationsPath

	fmt.Printf("\nTrying to create migrate instance with:\n")
	fmt.Printf("  migrations URL: %s\n", migrationURL)
	fmt.Printf("  database URL: %s\n", databaseURL)

	m, err := migrate.New(
		migrationURL,
		databaseURL,
	)
	if err != nil {
		fmt.Printf("ERROR: failed to create migrate instance: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\nRunning migrations...")
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		fmt.Printf("ERROR: failed to run migrations: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("SUCCESS: migrations completed!")
}
