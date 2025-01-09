package main

import (
  "bytes"
  "context"
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

  "go.opentelemetry.io/otel"
  //"go.opentelemetry.io/otel/attribute"
  "go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	//"go.opentelemetry.io/otel/trace"
)

var db *sql.DB


var tp *sdktrace.TracerProvider

//go:embed static/*
var staticFiles embed.FS

// initTracer creates and registers trace provider instance.
func initTracer() error {
	exp, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		return fmt.Errorf("failed to initialize stdouttrace exporter: %w", err)
	}
	bsp := sdktrace.NewBatchSpanProcessor(exp)
	tp = sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSpanProcessor(bsp),
	)
	otel.SetTracerProvider(tp)
	return nil
}

func populateDatabase(filename string) error {
	data, err := staticFiles.ReadFile(filename)
	if err != nil {
		log.Printf("Failed to open SQL file: %v", err)
		return err
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
  return nil
}

func initializeDatabase() error {
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

func tipCount() (int, error) {
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM tips").Scan(&count); err != nil {
    return 0, err
  }
	return count, nil
}

func dailyTipHandler(w http.ResponseWriter, r *http.Request) {
	today := time.Now().Day()

	count, err := tipCount()
	if err != nil {
		http.Error(w, "Failed to fetch tip count", http.StatusInternalServerError)
		return
	}

	dailyTipID := count % today
	var tip string
	err = db.QueryRow("SELECT tip FROM tips WHERE id = ?", dailyTipID).Scan(&tip)
	if err != nil {
		http.Error(w, "Failed to fetch daily tip", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(`{"tip": "%s"}`, tip)))
}

func randomTipHandler(w http.ResponseWriter, r *http.Request) {
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

  tracer := otel.Tracer("go.opentelemetry.io/contrib/examples/namedtracer")
  _, span := tracer.Start(r.Context(), "Sub operation...")
	defer span.End()

	span.AddEvent("Sub span event")

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
  // initialize trace provider.
	if err := initTracer(); err != nil {
		log.Panic(err)
	}

	// Create a named tracer with package path as its name.
	//tracer := tp.Tracer("vimtricks")
	ctx := context.Background()
	defer func() { _ = tp.Shutdown(ctx) }()

  //var span trace.Span
	//ctx, span = tracer.Start(ctx, "operation")
	//defer span.End()

  if err := initializeDatabase(); err != nil {
    log.Fatalf("DB initialization failed: %v", err)
  }

  if err := populateDatabase("static/populate.sql"); err != nil {
      panic(err)
    }
	defer db.Close()

	//populateDatabase("static/populate.sql")

	router := httprouter.New()
	router.Handler(http.MethodGet, "/static/*filepath", http.FileServer(http.FS(staticFiles)))
	router.HandlerFunc(http.MethodGet, "/", indexHandler)
	router.HandlerFunc(http.MethodGet, "/daily-tip", dailyTipHandler)
	router.HandlerFunc(http.MethodGet, "/random-tip", randomTipHandler)

	// Start server
	log.Println("Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}
