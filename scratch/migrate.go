package scratch

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"sort"
	"strings"

	_ "modernc.org/sqlite"
)

func Migrate() {
	db, err := sql.Open("sqlite", "databaseGotBuffed.db")
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	// Find all migration files
	files, err := filepath.Glob("internal/db/migrations/*.up.sql")
	if err != nil {
		log.Fatalf("failed to glob migrations: %v", err)
	}
	sort.Strings(files)

	fmt.Println("Applying pending migrations to databaseGotBuffed.db...")
	for _, file := range files {
		fmt.Printf("Processing %s...\n", file)
		content, err := ioutil.ReadFile(file)
		if err != nil {
			log.Fatalf("failed to read file %s: %v", file, err)
		}

		// Execute statements
		queries := strings.Split(string(content), ";")
		for _, query := range queries {
			trimmed := strings.TrimSpace(query)
			if trimmed == "" {
				continue
			}

			_, err := db.Exec(trimmed)
			if err != nil {
				// Ignore errors like "already exists" or duplicate columns
				if strings.Contains(err.Error(), "already exists") || strings.Contains(err.Error(), "duplicate column") {
					continue
				}
				fmt.Printf("  Warning/Error in statement [%s]: %v\n", trimmed, err)
			}
		}
	}
	fmt.Println("Database migrations updated completely!")
}
