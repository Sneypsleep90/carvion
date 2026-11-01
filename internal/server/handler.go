package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/Sneypsleep90/carvion/internal/core/domain"
	"github.com/go-chi/chi/v5"
)

const serviceName = "carvion-api"

type Pinger interface {
	Ping(context.Context) error
}

type Core interface {
	ListAdapters(context.Context) ([]domain.AdapterInfo, error)
	CheckCompatibility(context.Context, string, domain.CompatibilityRequest) (domain.CompatibilityResult, error)
}

func NewHandler(pinger Pinger, logger *slog.Logger, core Core) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	router := chi.NewRouter()
	router.Use(requestLogger(logger))
	router.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, response{Status: "ok", Service: serviceName})
	})
	router.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if pinger == nil {
			writeJSON(w, http.StatusServiceUnavailable, response{Status: "unavailable", Service: serviceName})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := pinger.Ping(ctx); err != nil {
			logger.ErrorContext(r.Context(), "database readiness check failed", "error", err)
			writeJSON(w, http.StatusServiceUnavailable, response{Status: "unavailable", Service: serviceName})
			return
		}
		writeJSON(w, http.StatusOK, response{Status: "ready", Service: serviceName})
	})
	router.Get("/api/v1/adapters", adaptersListHandler(core))
	router.Post("/api/v1/compatibility/check", compatibilityCheckHandler(core))
	return router
}

type response struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func requestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			statusWriter := &statusWriter{ResponseWriter: w}
			next.ServeHTTP(statusWriter, r)
			status := statusWriter.status
			if status == 0 {
				status = http.StatusOK
			}
			logger.Info("http request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", status),
				slog.Duration("duration", time.Since(start)),
			)
		})
	}
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(body)
}
