//go:build integration

package database

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestConnect(t *testing.T) {
	databaseURL := os.Getenv("CARVION_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set CARVION_TEST_DATABASE_URL to run the PostgreSQL integration test")
	}

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	pool, err := Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("pool.Ping() error = %v", err)
	}
}
