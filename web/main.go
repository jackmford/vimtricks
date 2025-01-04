package main

import (
  "bytes"
	"database/sql"
	"embed"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"bufio"

	"github.com/julienschmidt/httprouter"
	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

//go:embed static/*
var staticFiles embed.FS

func executeSQLFile(filename string) {
	data, err := staticFiles.ReadFile("static/populate.sql")
	if err != nil {
		log.Println("Failed to open SQL file:", err)
		return
	}

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		query := scanner.Text()
		if query != "" {
			_, err = db.Exec(query)
			if err != nil {
				log.Fatal("Failed to execute query:", query, "Error:", err)
			}
		}
	}
}

func initializeDatabase() error {
	var err error
	db, err = sql.Open("sqlite3", "./vimtips.db")
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}

	createTableQuery := `
	CREATE TABLE IF NOT EXISTS tips (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		tip TEXT NOT NULL
	);
	`
	_, err = db.Exec(createTableQuery)
	if err != nil {
		log.Fatal("Failed to create table:", err)
	}
	return db.Ping()
}

func tipCount() (int, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM tips").Scan(&count)
	return count, err
}

func dailyTip(w http.ResponseWriter, r *http.Request) {
	today := time.Now().Day()

	count, err := tipCount()
	if err != nil {
		http.Error(w, "Failed to fetch tip count", http.StatusInternalServerError)
		return
	}

	dailyTip := today % count
	var tip string
	err = db.QueryRow("SELECT tip FROM tips WHERE id = ?", dailyTip).Scan(&tip)
	if err != nil {
		http.Error(w, "Failed to fetch daily tip", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(`{"tip": "%s"}`, tip)))
}

func randomTip(w http.ResponseWriter, r *http.Request) {
	rand.Seed(time.Now().UnixNano())

	count, err := tipCount()
	if err != nil {
		http.Error(w, "Failed to fetch tip count", http.StatusInternalServerError)
		return
	}

	randomID := rand.Intn(count)
	var tip string
	err = db.QueryRow("SELECT tip FROM tips WHERE id = ?", randomID).Scan(&tip)
	if err != nil {
		http.Error(w, "Failed to fetch random tip", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(`{"tip": "%s"}`, tip)))
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	content, err := staticFiles.ReadFile("static/index.html")
	if err != nil {
		http.Error(w, "Failed to load index.html", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write(content)
}

func main() {
  if err := initializeDatabase(); err != nil {
    log.Fatalf("DB initialization failed: %v", err)
  }
	defer db.Close()

	executeSQLFile("static/populate.sql")

	router := httprouter.New()
	router.Handler(http.MethodGet, "/static/*filepath", http.FileServer(http.FS(staticFiles)))
	router.HandlerFunc(http.MethodGet, "/", indexHandler)
	router.HandlerFunc(http.MethodGet, "/daily-tip", dailyTip)
	router.HandlerFunc(http.MethodGet, "/random-tip", randomTip)

	// Start server
	log.Println("Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}
