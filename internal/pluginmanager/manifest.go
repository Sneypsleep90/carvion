package pluginmanager

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	sdk "github.com/Sneypsleep90/carvion/pkg/adaptersdk/v1"
)

const maxManifestBytes = 64 << 10

var (
	adapterIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)
	semverPattern    = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$`)
)

func readManifest(pluginDirectory string) (sdk.AdapterManifest, error) {
	path := filepath.Join(pluginDirectory, "manifest.json")
	info, err := os.Lstat(path)
	if err != nil {
		return sdk.AdapterManifest{}, fmt.Errorf("stat manifest: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() > maxManifestBytes {
		return sdk.AdapterManifest{}, errors.New("manifest must be a regular file under 64 KiB")
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		return sdk.AdapterManifest{}, fmt.Errorf("read manifest: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.DisallowUnknownFields()
	var manifest sdk.AdapterManifest
	if err := decoder.Decode(&manifest); err != nil {
		return sdk.AdapterManifest{}, fmt.Errorf("decode manifest: %w", err)
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return sdk.AdapterManifest{}, errors.New("manifest must contain exactly one JSON object")
	}
	if err := validateManifest(manifest); err != nil {
		return sdk.AdapterManifest{}, err
	}
	return manifest, nil
}

func validateManifest(manifest sdk.AdapterManifest) error {
	if manifest.ManifestVersion != sdk.ManifestVersion {
		return fmt.Errorf("unsupported manifest version %d", manifest.ManifestVersion)
	}
	if manifest.APIVersion != sdk.APIVersion {
		return fmt.Errorf("unsupported adapter API version %q", manifest.APIVersion)
	}
	if !adapterIDPattern.MatchString(manifest.ID) {
		return errors.New("adapter ID must contain lowercase letters, digits, and hyphens")
	}
	if name := strings.TrimSpace(manifest.Name); name == "" || len(name) > 128 {
		return errors.New("adapter name must be between 1 and 128 characters")
	}
	if !semverPattern.MatchString(manifest.Version) {
		return errors.New("adapter version must use semantic versioning")
	}
	if strings.TrimSpace(manifest.Executable) == "" {
		return errors.New("adapter executable is required")
	}
	if !manifest.Capabilities.Detect && !manifest.Capabilities.CheckCompatibility && !manifest.Capabilities.InspectResources {
		return errors.New("adapter must declare at least one capability")
	}
	return nil
}

func resolveExecutable(pluginDirectory, executable string) (string, error) {
	if filepath.IsAbs(executable) {
		return "", errors.New("adapter executable must be relative to its plugin directory")
	}
	relative := filepath.Clean(filepath.FromSlash(executable))
	if relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", errors.New("adapter executable must stay within its plugin directory")
	}

	pluginRoot, err := filepath.EvalSymlinks(pluginDirectory)
	if err != nil {
		return "", fmt.Errorf("resolve plugin directory: %w", err)
	}
	candidate := filepath.Join(pluginRoot, relative)
	resolved, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return "", fmt.Errorf("resolve adapter executable: %w", err)
	}
	inside, err := isWithin(pluginRoot, resolved)
	if err != nil || !inside {
		return "", errors.New("adapter executable resolves outside its plugin directory")
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", fmt.Errorf("stat adapter executable: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
		return "", errors.New("adapter executable must be an executable regular file")
	}
	return resolved, nil
}

func isWithin(root, path string) (bool, error) {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return false, err
	}
	return relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)), nil
}
