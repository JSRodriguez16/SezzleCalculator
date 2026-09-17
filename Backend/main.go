package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"sezzlecalculator/backend/business"
	"sezzlecalculator/backend/presentation"
)

var processExit = os.Exit
var notifyContext = signal.NotifyContext
var listenAndServe = func(server *http.Server) error { return server.ListenAndServe() }
var shutdownServer = func(server *http.Server, ctx context.Context) error { return server.Shutdown(ctx) }

func main() {
	if err := run(); err != nil {
		slog.Error("server stopped", "error", err)
		processExit(1)
	}
}

func run() error {
	ctx, stop := notifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return runWithContext(ctx)
}

func runWithContext(ctx context.Context) error {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	number, err := strconv.Atoi(port)
	if err != nil || number < 1 || number > 65535 {
		return fmt.Errorf("PORT must be an integer between 1 and 65535")
	}
	handler, err := presentation.NewHandler(business.Service{}, os.Getenv("STATIC_DIR"))
	if err != nil {
		return fmt.Errorf("configure static frontend: %w", err)
	}
	server := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	errCh := make(chan error, 1)
	go func() {
		slog.Info("calculator listening", "address", server.Addr)
		errCh <- listenAndServe(server)
	}()
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		slog.Info("shutting down calculator")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := shutdownServer(server, shutdownCtx); err != nil {
			server.Close()
			return fmt.Errorf("graceful shutdown: %w", err)
		}
		if err := <-errCh; !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
}
