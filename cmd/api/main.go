package main

import (
	"context"
	"errors"
	"jev/config"
	"jev/internal/database"
	"jev/internal/httpapi"
	"jev/internal/observability"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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

	connectContext, connectCancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer connectCancel()

	if err := database.ConnectDB(db, connectContext); err != nil {
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

	if cfg.IsDevelop {
		go observability.StartPprof(logger, cfg.PprofAddress)
	}

	signalContext, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	serverError := make(chan error, 1)

	go func() {
		logger.Info("API server listening", "address", cfg.APIAddress)
		serverError <- server.ListenAndServe()
	}()

	select {
	case err := <-serverError:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("API server stopped unexpectedly", "error", err)
		}
		return
	case <-signalContext.Done():
		logger.Info("API server shutdown requested")
	}

	shutdownContext, shutdownCancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownContext); err != nil {
		logger.Error("failed to gracefully shut down API server", "error", err)
		if closeErr := server.Close(); closeErr != nil {
			logger.Error("failed to force close API server", "error", closeErr)
		}
		return
	}

	logger.Info("API server shutdown completed")
}
