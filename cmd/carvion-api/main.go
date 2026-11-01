package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/Sneypsleep90/carvion/internal/composition"
	"github.com/Sneypsleep90/carvion/internal/config"
	"github.com/Sneypsleep90/carvion/internal/server"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("application stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	coreConfig, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}
	httpConfig, err := config.LoadHTTP()
	if err != nil {
		return fmt.Errorf("load HTTP configuration: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	runtime, err := composition.NewLocal(ctx, coreConfig, logger)
	if err != nil {
		return fmt.Errorf("build application runtime: %w", err)
	}
	defer runtime.Close()

	address := net.JoinHostPort(httpConfig.Host, strconv.Itoa(httpConfig.Port))
	httpServer := &http.Server{
		Addr:              address,
		Handler:           server.NewHandler(runtime, logger, runtime.Core),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- httpServer.ListenAndServe()
	}()

	logger.Info("HTTP server listening", "address", address, "environment", coreConfig.AppEnv)

	select {
	case err := <-serverErrors:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve HTTP: %w", err)
		}
		return nil
	case <-ctx.Done():
		shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelShutdown()

		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			_ = httpServer.Close()
			return fmt.Errorf("shut down HTTP server: %w", err)
		}

		if err := <-serverErrors; err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve HTTP: %w", err)
		}

		logger.Info("HTTP server shut down")
		return nil
	}
}
