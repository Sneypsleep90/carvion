package composition

import (
	"context"
	"errors"
	"testing"

	"github.com/Sneypsleep90/carvion/internal/config"
)

func TestNewLocalStopsBeforeOpeningDependenciesWhenContextIsCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := NewLocal(ctx, config.Config{}, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("NewLocal() error = %v, want context.Canceled", err)
	}
}

func TestRuntimeCloseAndPingHandleNilRuntime(t *testing.T) {
	var runtime *Runtime
	runtime.Close()
	if err := runtime.Ping(context.Background()); err == nil {
		t.Fatal("Ping() error = nil, want unavailable runtime error")
	}
}
