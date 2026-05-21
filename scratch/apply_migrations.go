package scratch

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	_ "modernc.org/sqlite"
)

func ApplyMigrations() {
	db, err := sql.Open("sqlite", "databaseGotBuffed.db")
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	// Apply migration 000006_giveaways_enhancements.up.sql
	migrationFile := "internal/db/migrations/000006_giveaways_enhancements.up.sql"
	content, err := ioutil.ReadFile(migrationFile)
	if err != nil {
		log.Fatalf("failed to read migration file %s: %v", migrationFile, err)
	}

	fmt.Printf("Executing migration:\n%s\n", string(content))
	_, err = db.Exec(string(content))
	if err != nil {
		fmt.Printf("Migration warning/error (might already exist): %v\n", err)
		os.Exit(0)
	}

	fmt.Println("Migration executed successfully!")
}
