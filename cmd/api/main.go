package main

import (
	"context"
	"jev/config"
	"jev/internal/database"
	"jev/internal/httpapi"
	"jev/internal/observability"
	"log"
	"net/http"
	"time"

	applog "jev/internal/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Printf("failed to load application configuration: %v", err)
		return
	}

	logger := applog.New()

	db, err := database.NewDB(cfg.DBDSN)
	if err != nil {
		logger.Error("failed to open database", "error", err)
		return
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Error("failed to close database", "error", err)
		}
	}()

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	if err := database.ConnectDB(db, ctx); err != nil {
		logger.Error("failed to connect to database", "error", err)
		return
	}

	router := httpapi.NewRouter(db, logger)

	server := http.Server{
		Addr:              cfg.APIAddress,
		Handler:           router,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	go observability.StartPprof(logger, cfg.PprofAddress)

	logger.Info("API server listening", "address", cfg.APIAddress)
	if err := server.ListenAndServe(); err != nil {
		logger.Error("API server stopped", "error", err)
	}
}
