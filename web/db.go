package main

import (
  "bytes"
	"database/sql"
	"fmt"
	"log"

	"bufio"

	_ "github.com/mattn/go-sqlite3"
)


func PopulateDatabase(filename string) {
	data, err := staticFiles.ReadFile(filename)
	if err != nil {
		log.Printf("Failed to open SQL file: %v", err)
		return
	}

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		query := scanner.Text()
		if query != "" {
			if _, err = db.Exec(query); err != nil {
				log.Fatalf("Failed to execute query: %v", err)
			}
		}
	}
}

func InitializeDatabase() error {
	var err error
	db, err = sql.Open("sqlite3", "./vimtips.db"); if err != nil {
    return fmt.Errorf("Failed to open database: %v", err)
  }

	createTableQuery := `
	CREATE TABLE IF NOT EXISTS tips (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		tip TEXT NOT NULL
	);
	`

	if _, err = db.Exec(createTableQuery); err != nil {
    return fmt.Errorf("Failed to create database table: %v", err)
	}

	return db.Ping()
}
