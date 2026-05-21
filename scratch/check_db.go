package scratch

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

func CheckDB() {
	db, err := sql.Open("sqlite", "databaseGotBuffed.db")
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table'")
	if err != nil {
		log.Fatalf("failed to query tables: %v", err)
	}
	defer rows.Close()

	fmt.Println("Tables in databaseGotBuffed.db:")
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			log.Fatalf("failed to scan: %v", err)
		}
		fmt.Printf("- %s\n", name)
	}
}
