package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type pingerFunc func(context.Context) error

func (f pingerFunc) Ping(ctx context.Context) error {
	return f(ctx)
}

func TestHealthzReturnsJSONOK(t *testing.T) {
	handler := NewHandler(pingerFunc(func(context.Context) error {
		t.Fatal("/healthz should not ping the database")
		return nil
	}), slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)), nil)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", contentType)
	}
	assertResponse(t, recorder, "ok")
}

func TestReadyzReturnsReadyWhenDatabasePingSucceeds(t *testing.T) {
	var pingSawDeadline bool
	handler := NewHandler(pingerFunc(func(ctx context.Context) error {
		deadline, ok := ctx.Deadline()
		pingSawDeadline = ok && time.Until(deadline) > 0 && time.Until(deadline) <= 2*time.Second
		return nil
	}), slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)), nil)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if !pingSawDeadline {
		t.Error("database ping context did not have a deadline within 2 seconds")
	}
	assertResponse(t, recorder, "ready")
}

func TestReadyzReturnsUnavailableWithoutLeakingDatabaseError(t *testing.T) {
	databaseErr := errors.New("database password rejected")
	handler := NewHandler(pingerFunc(func(context.Context) error { return databaseErr }),
		slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)), nil)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
	assertResponse(t, recorder, "unavailable")
	if strings.Contains(recorder.Body.String(), databaseErr.Error()) {
		t.Errorf("response exposed database error: %q", recorder.Body.String())
	}
}

func TestReadyzLogsDatabasePingFailure(t *testing.T) {
	var logs bytes.Buffer
	databaseErr := errors.New("connection refused")
	handler := NewHandler(pingerFunc(func(context.Context) error { return databaseErr }),
		slog.New(slog.NewJSONHandler(&logs, nil)), nil)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)

	handler.ServeHTTP(recorder, request)

	if !strings.Contains(logs.String(), "database readiness check failed") {
		t.Errorf("logs = %q, want readiness failure message", logs.String())
	}
	if !strings.Contains(logs.String(), databaseErr.Error()) {
		t.Errorf("logs = %q, want database error details", logs.String())
	}
}

func TestRequestsProduceStructuredLogs(t *testing.T) {
	var logs bytes.Buffer
	handler := NewHandler(pingerFunc(func(context.Context) error { return nil }),
		slog.New(slog.NewJSONHandler(&logs, nil)), nil)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	handler.ServeHTTP(recorder, request)

	var record map[string]any
	if err := json.Unmarshal(logs.Bytes(), &record); err != nil {
		t.Fatalf("log is not JSON: %v; log = %q", err, logs.String())
	}
	for key, want := range map[string]any{
		"method": http.MethodGet,
		"path":   "/healthz",
		"status": float64(http.StatusOK),
	} {
		got, ok := record[key]
		if !ok {
			t.Errorf("log missing %q field: %v", key, record)
			continue
		}
		if got != want {
			t.Errorf("log %s = %v, want %v", key, got, want)
		}
	}
	duration, ok := record["duration"]
	if !ok || duration == nil {
		t.Errorf("log missing duration field: %v", record)
	} else if duration, ok := duration.(float64); !ok || duration < 0 {
		t.Errorf("log duration = %v, want a non-negative duration in nanoseconds", duration)
	}
}

func assertResponse(t *testing.T, recorder *httptest.ResponseRecorder, wantStatus string) {
	t.Helper()
	var body struct {
		Status  string `json:"status"`
		Service string `json:"service"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response JSON: %v; body = %q", err, recorder.Body.String())
	}
	if body.Status != wantStatus {
		t.Errorf("response status = %q, want %q", body.Status, wantStatus)
	}
	if body.Service != "carvion-api" {
		t.Errorf("response service = %q, want %q", body.Service, "carvion-api")
	}
}
