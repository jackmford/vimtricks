package main

import (
  "database/sql"
  "embed"
	"fmt"
  "net/http"
	"time"
  "math/rand"

  "bufio"
  "os"
  "log"

	"github.com/gin-gonic/gin"
  _ "github.com/mattn/go-sqlite3"
)

var db *sql.DB
var staticFiles embed.FS

func executeSQLFile(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		log.Println("Failed to open SQL file:", err)
		return
	}
	defer file.Close()


	scanner := bufio.NewScanner(file)
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

func initDatabase() error {
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

func dailyTip(c *gin.Context) {
  today := time.Now().Day()
  var err error

  count, err := tipCount()
  if err != nil {
    fmt.Errorf("failed to fetch tip: %w", err)
    return
  }	

  dailyTip := today%count
  var tip string
  err = db.QueryRow("SELECT tip FROM tips WHERE id = ?", dailyTip).Scan(&tip)
  if err != nil {
    fmt.Errorf("failed to fetch tip: %w", err)
  }
  c.JSON(http.StatusOK, gin.H{
    "tip": tip,
  })
}

func randomTip(c *gin.Context) {
	rand.Seed(time.Now().UnixNano())

  var err error
  count, err := tipCount()
  if err != nil {
    fmt.Errorf("failed to fetch tip: %w", err)
    return
  }	

  fmt.Println(count)
  randomId := rand.Intn(count)
  fmt.Println(randomId)
  var tip string
  err = db.QueryRow("SELECT tip FROM tips WHERE id = ?", randomId).Scan(&tip)
  if err != nil {
    fmt.Errorf("failed to fetch tip: %w", err)
  }
  c.JSON(http.StatusOK, gin.H{
    "tip": tip,
  })
}

func main() {

  err := initDatabase()
  if err != nil {
    log.Fatal(err)
  }
	defer db.Close()
  executeSQLFile("./web/populate.sql")

  r := gin.Default()
  r.StaticFS("/static", http.FS(staticFiles))
  //r.Static("/static", "./static")

  r.GET("/", func(c *gin.Context) {
    c.File("static/index.html")
  })

	// Route to get the daily tip
	r.GET("/daily-tip", dailyTip)
	// Route to get a random tip
	r.GET("/random-tip", randomTip)


	// Start server
	r.Run(":8080")
}

