package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"jev/handler"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type HealthResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := HealthResponse{
		Status:  "ok",
		Message: "hello world",
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("failed to encode health response: %v", err)
	}
}

func GetDSN() string {
	dsn := os.Getenv("DB_DSN")
	return dsn
}

func NewDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func ConnectDB(db *sql.DB, ctx context.Context) error {
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(3 * time.Minute)

	return db.PingContext(ctx)
}

func main() {
	dsn := GetDSN()

	db, err := NewDB(dsn)

	if err != nil {
		log.Fatal("failed to open new db", err)
	}

	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := ConnectDB(db, ctx); err != nil {
		log.Fatal("failed to connect db", err)
	}

	logger := slog.New(
		slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			AddSource: false,
			Level: slog.LevelInfo,
		}),
	)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", health)

	mux.HandleFunc("POST /v1/users", handler.CreateUser(db, logger))

	log.Println("listening on http://localhost:8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
