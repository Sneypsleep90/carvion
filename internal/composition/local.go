package composition

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/Sneypsleep90/carvion/internal/config"
	"github.com/Sneypsleep90/carvion/internal/core/application"
	"github.com/Sneypsleep90/carvion/internal/core/infrastructure/adapterbridge"
	"github.com/Sneypsleep90/carvion/internal/core/infrastructure/postgres"
	"github.com/Sneypsleep90/carvion/internal/database"
	"github.com/Sneypsleep90/carvion/internal/pluginmanager"
	"github.com/jackc/pgx/v5/pgxpool"
)

const databaseStartupTimeout = 10 * time.Second

// Runtime owns the local development dependencies shared by the application
// core and its transport adapters.
type Runtime struct {
	Core *application.Service
	pool *pgxpool.Pool
}

// NewLocal constructs the core with local PostgreSQL and trusted local adapter
// processes. A Wails entry point can call Core directly without starting HTTP.
func NewLocal(ctx context.Context, cfg config.Config, logger *slog.Logger) (*Runtime, error) {
	if ctx == nil {
		return nil, errors.New("runtime context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if logger == nil {
		logger = slog.Default()
	}

	databaseCtx, cancel := context.WithTimeout(ctx, databaseStartupTimeout)
	pool, err := database.Connect(databaseCtx, cfg.DatabaseURL)
	cancel()
	if err != nil {
		return nil, fmt.Errorf("connect local PostgreSQL: %w", err)
	}

	manager, err := pluginmanager.DiscoverContext(ctx, cfg.AdaptersDir, pluginmanager.DefaultCallTimeout, logger)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("discover local adapters: %w", err)
	}

	core, err := application.NewServiceWithLocalization(
		adapterbridge.New(manager),
		postgres.NewCatalogRepository(pool),
		postgres.NewLocalizationRepository(pool),
	)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("build application core: %w", err)
	}
	return &Runtime{Core: core, pool: pool}, nil
}

func (r *Runtime) Ping(ctx context.Context) error {
	if r == nil || r.pool == nil {
		return errors.New("runtime database is unavailable")
	}
	return r.pool.Ping(ctx)
}

func (r *Runtime) Close() {
	if r != nil && r.pool != nil {
		r.pool.Close()
	}
}
