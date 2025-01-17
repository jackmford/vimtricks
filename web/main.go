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
	//"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	//"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
)

var db *sql.DB


var tp *sdktrace.TracerProvider

var tracer trace.Tracer

//go:embed static/*
var staticFiles embed.FS


// initTracer creates and registers trace provider instance.
func initTracer(ctx context.Context) (trace.Tracer, error) {
	//exp, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
  exp, err := otlptracehttp.New(ctx,
    otlptracehttp.WithEndpoint("localhost:4318"),
    otlptracehttp.WithInsecure(),
    )

	if err != nil {
		return nil, fmt.Errorf("failed to initialize stdouttrace exporter: %w", err)
	}
	bsp := sdktrace.NewBatchSpanProcessor(exp)
	tp = sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSpanProcessor(bsp),
	)
	otel.SetTracerProvider(tp)
  tracer = otel.Tracer("github.com/jackmford/vimtricks")
	return tracer, nil
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

func initializeDatabase(ctx context.Context) error {
  _, span := tracer.Start(ctx, "Sub operation...")
	defer span.End()

	span.AddEvent("Init db event")


	var err error
	db, err = sql.Open("sqlite3", "./vimtips.db"); if err != nil {
    span.RecordError(err)
    return fmt.Errorf("Failed to open database: %v", err)
  }

	createTableQuery := `
	CREATE TABLE IF NOT EXISTS tips (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		tip TEXT NOT NULL
	);
	`

	if _, err = db.Exec(createTableQuery); err != nil {
    span.RecordError(err)
    return fmt.Errorf("Failed to create database table: %v", err)
	}

	return db.Ping()
}

func tipCount(ctx context.Context) (int, error) {
  ctx, childSpan := tracer.Start(ctx, "tipCount retrieval")
  defer childSpan.End()

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM tips").Scan(&count); err != nil {
    return 0, err
  }
	return count, nil
}

func dailyTipHandler(w http.ResponseWriter, r *http.Request) {
  ctx, span := tracer.Start(r.Context(), "Daily tip operation")
	defer span.End()

	span.AddEvent("Daily tip event")

	today := time.Now().Day()

	count, err := tipCount(ctx)
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

	count, err := tipCount(r.Context())
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
  _, span := tracer.Start(r.Context(), "Index handler...")
	defer span.End()

	span.AddEvent("Index Handler")

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
	ctx := context.Background()
	defer func() { _ = tp.Shutdown(ctx) }()

	if _, err := initTracer(ctx); err != nil {
		log.Panic(err)
	}


  if err := initializeDatabase(ctx); err != nil {
    log.Fatalf("DB initialization failed: %v", err)
  }

  if err := populateDatabase("static/populate.sql"); err != nil {
      panic(err)
    }
	defer db.Close()

	router := httprouter.New()
	router.Handler(http.MethodGet, "/static/*filepath", http.FileServer(http.FS(staticFiles)))
	router.HandlerFunc(http.MethodGet, "/", indexHandler)
	router.HandlerFunc(http.MethodGet, "/daily-tip", dailyTipHandler)
	router.HandlerFunc(http.MethodGet, "/random-tip", randomTipHandler)

	// Start server
	log.Println("Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}
