package pluginmanager

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"time"

	sdk "github.com/Sneypsleep90/carvion/pkg/adaptersdk/v1"
)

const (
	DefaultCallTimeout = 3 * time.Second
	maxConcurrentCalls = 4
)

var (
	ErrAdapterNotFound       = errors.New("adapter not found")
	ErrCapabilityUnavailable = errors.New("adapter capability is not declared")
	ErrPluginUnavailable     = errors.New("adapter process is unavailable")
	ErrPluginTimeout         = errors.New("adapter call timed out")
)

type AdapterSummary struct {
	ID           string                  `json:"id"`
	Name         string                  `json:"name"`
	Version      string                  `json:"version"`
	APIVersion   string                  `json:"api_version"`
	Capabilities sdk.AdapterCapabilities `json:"capabilities"`
}

type plugin struct {
	manifest   sdk.AdapterManifest
	directory  string
	executable string
	slots      chan struct{}
}

type Manager struct {
	plugins map[string]*plugin
	timeout time.Duration
	logger  *slog.Logger
}

// Discover reads manifests from direct child directories of a trusted local
// catalog. Invalid plugins are logged and skipped; a missing catalog is empty.
func Discover(root string, timeout time.Duration, logger *slog.Logger) (*Manager, error) {
	return DiscoverContext(context.Background(), root, timeout, logger)
}

// DiscoverContext scans a trusted local catalog and stops promptly when the
// startup context is canceled.
func DiscoverContext(ctx context.Context, root string, timeout time.Duration, logger *slog.Logger) (*Manager, error) {
	if ctx == nil {
		return nil, errors.New("adapter discovery context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if root == "" {
		return nil, errors.New("adapter catalog directory is required")
	}
	if timeout <= 0 {
		timeout = DefaultCallTimeout
	}
	if logger == nil {
		logger = slog.Default()
	}
	manager := &Manager{plugins: make(map[string]*plugin), timeout: timeout, logger: logger}

	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve adapter catalog path: %w", err)
	}
	rootInfo, err := os.Stat(absoluteRoot)
	if errors.Is(err, os.ErrNotExist) {
		logger.Info("adapter catalog does not exist", "directory", absoluteRoot)
		return manager, nil
	}
	if err != nil {
		return nil, fmt.Errorf("stat adapter catalog: %w", err)
	}
	if !rootInfo.IsDir() {
		return nil, fmt.Errorf("adapter catalog %q is not a directory", absoluteRoot)
	}
	catalog, err := filepath.EvalSymlinks(absoluteRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve adapter catalog: %w", err)
	}
	entries, err := os.ReadDir(catalog)
	if err != nil {
		return nil, fmt.Errorf("read adapter catalog: %w", err)
	}

	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if !entry.IsDir() {
			continue
		}
		directory := filepath.Join(catalog, entry.Name())
		info, err := os.Lstat(directory)
		if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			logger.Warn("adapter discovery skipped", "directory", entry.Name(), "error", "plugin entry is not a real directory")
			continue
		}

		manifest, err := readManifest(directory)
		if err != nil {
			logger.Warn("adapter discovery skipped", "directory", entry.Name(), "error", err)
			continue
		}
		if _, exists := manager.plugins[manifest.ID]; exists {
			logger.Warn("adapter discovery skipped", "directory", entry.Name(), "adapter_id", manifest.ID, "error", "duplicate adapter ID")
			continue
		}
		executable, err := resolveExecutable(directory, manifest.Executable)
		if err != nil {
			logger.Warn("adapter discovery skipped", "directory", entry.Name(), "adapter_id", manifest.ID, "error", err)
			continue
		}

		registered := &plugin{
			manifest:   manifest,
			directory:  directory,
			executable: executable,
			slots:      make(chan struct{}, maxConcurrentCalls),
		}
		manager.plugins[manifest.ID] = registered
		var pluginInfo sdk.AdapterManifest
		if err := manager.invoke(ctx, registered, "GetInfo", struct{}{}, &pluginInfo); err != nil {
			if contextErr := ctx.Err(); contextErr != nil {
				return nil, contextErr
			}
			delete(manager.plugins, manifest.ID)
			logger.Warn("adapter registration skipped", "directory", entry.Name(), "adapter_id", manifest.ID, "error", err)
			continue
		}
		if pluginInfo != manifest {
			delete(manager.plugins, manifest.ID)
			logger.Warn("adapter registration skipped", "directory", entry.Name(), "adapter_id", manifest.ID, "error", "GetInfo does not match manifest")
			continue
		}
		logger.Info("adapter registered", "adapter_id", manifest.ID, "version", manifest.Version, "api_version", manifest.APIVersion)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return manager, nil
}

func (m *Manager) List() []AdapterSummary {
	items := make([]AdapterSummary, 0, len(m.plugins))
	for _, registered := range m.plugins {
		manifest := registered.manifest
		items = append(items, AdapterSummary{
			ID:           manifest.ID,
			Name:         manifest.Name,
			Version:      manifest.Version,
			APIVersion:   manifest.APIVersion,
			Capabilities: manifest.Capabilities,
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items
}

func (m *Manager) GetInfo(ctx context.Context, adapterID string) (sdk.AdapterManifest, error) {
	registered, err := m.getPlugin(adapterID)
	if err != nil {
		return sdk.AdapterManifest{}, err
	}
	var result sdk.AdapterManifest
	if err := m.invoke(ctx, registered, "GetInfo", struct{}{}, &result); err != nil {
		return sdk.AdapterManifest{}, err
	}
	return result, nil
}

func (m *Manager) Detect(ctx context.Context, adapterID string, request sdk.DetectRequest) (sdk.DetectionResult, error) {
	registered, err := m.getPlugin(adapterID)
	if err != nil {
		return sdk.DetectionResult{}, err
	}
	if !registered.manifest.Capabilities.Detect {
		return sdk.DetectionResult{}, ErrCapabilityUnavailable
	}
	var result sdk.DetectionResult
	if err := m.invoke(ctx, registered, "Detect", request, &result); err != nil {
		return sdk.DetectionResult{}, err
	}
	if !validStatus(result.Status) {
		return sdk.DetectionResult{}, fmt.Errorf("%w: adapter returned an invalid detection status", ErrPluginUnavailable)
	}
	return result, nil
}

func (m *Manager) CheckCompatibility(ctx context.Context, adapterID string, request sdk.CompatibilityRequest) (sdk.CompatibilityResult, error) {
	registered, err := m.getPlugin(adapterID)
	if err != nil {
		return sdk.CompatibilityResult{}, err
	}
	if !registered.manifest.Capabilities.CheckCompatibility {
		return sdk.CompatibilityResult{}, ErrCapabilityUnavailable
	}
	var result sdk.CompatibilityResult
	if err := m.invoke(ctx, registered, "CheckCompatibility", request, &result); err != nil {
		return sdk.CompatibilityResult{}, err
	}
	if !validStatus(result.Status) {
		return sdk.CompatibilityResult{}, fmt.Errorf("%w: adapter returned an invalid compatibility status", ErrPluginUnavailable)
	}
	return result, nil
}

func (m *Manager) InspectResources(ctx context.Context, adapterID string, request sdk.InspectResourcesRequest) (sdk.ResourceInspection, error) {
	registered, err := m.getPlugin(adapterID)
	if err != nil {
		return sdk.ResourceInspection{}, err
	}
	if !registered.manifest.Capabilities.InspectResources {
		return sdk.ResourceInspection{}, ErrCapabilityUnavailable
	}
	var result sdk.ResourceInspection
	if err := m.invoke(ctx, registered, "InspectResources", request, &result); err != nil {
		return sdk.ResourceInspection{}, err
	}
	return result, nil
}

func (m *Manager) getPlugin(adapterID string) (*plugin, error) {
	registered, ok := m.plugins[adapterID]
	if !ok {
		return nil, ErrAdapterNotFound
	}
	return registered, nil
}

func (m *Manager) invoke(ctx context.Context, registered *plugin, method string, params, result any) error {
	callCtx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()
	select {
	case registered.slots <- struct{}{}:
		defer func() { <-registered.slots }()
	case <-callCtx.Done():
		return m.callContextError(ctx, callCtx, registered.manifest.ID, method, "waiting for adapter process")
	}

	stderr, err := runRPC(callCtx, registered, method, params, result)
	if err == nil {
		return nil
	}
	if callCtx.Err() != nil {
		return m.callContextError(ctx, callCtx, registered.manifest.ID, method, err.Error())
	}
	m.logger.ErrorContext(ctx, "adapter RPC failed", "adapter_id", registered.manifest.ID, "method", method, "error", err, "stderr", stderr)
	return fmt.Errorf("%w: %v", ErrPluginUnavailable, err)
}

func (m *Manager) callContextError(parent, callCtx context.Context, adapterID, method, detail string) error {
	if errors.Is(callCtx.Err(), context.DeadlineExceeded) {
		m.logger.WarnContext(parent, "adapter RPC timed out", "adapter_id", adapterID, "method", method)
		return fmt.Errorf("%w: %s", ErrPluginTimeout, detail)
	}
	return fmt.Errorf("%w: %s", ErrPluginUnavailable, detail)
}

func validStatus(status sdk.CompatibilityStatus) bool {
	return status == sdk.StatusSupported || status == sdk.StatusUnsupported || status == sdk.StatusUnknown
}
