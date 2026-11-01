# Vehicle Adapter SDK v1

## Package and versioning

Adapter authors import `github.com/Sneypsleep90/carvion/pkg/adaptersdk/v1`. The manifest declares `manifest_version: 1` and `api_version: "v1"`. The manager rejects other versions and checks that the process `GetInfo` response matches the manifest before registering the adapter.

The SDK defines `AdapterManifest`, `AdapterCapabilities`, `VehicleIdentity`, `FirmwareInfo`, `HardwareInfo`, `CompatibilityResult`, request/response types, and the `Adapter` interface. Compatibility and detection states are `supported`, `unsupported`, and `unknown`. A status of `unknown` means the adapter has no exact profile evidence and makes no support claim.

## Process protocol

Each operation runs in a fresh process. The manager starts the manifest executable directly, writes one JSON-RPC 2.0 request as one newline-terminated JSON object to stdin, closes stdin, reads one response frame from stdout, and waits for process exit. Adapter diagnostics belong on stderr; stdout is reserved for the RPC response.

Supported methods are `GetInfo`, `Detect`, `CheckCompatibility`, and `InspectResources`. The SDK server rejects unknown methods and unknown request fields. Request and response frames are limited to 1 MiB. The manager enforces a three-second default deadline and allows at most four concurrent calls per adapter.

Example request:

```json
{"jsonrpc":"2.0","id":"1","method":"CheckCompatibility","params":{"identity":{"manufacturer":"CARVION TEST LAB","model":"Android Demo Unit","platform_id":"android-demo-platform"},"hardware":{"platform_id":"android-demo-platform","revision":"rev-a"},"firmware":{"build_id":"demo-build-a-001","version":"1.0.0-demo"}}}
```

Example response:

```json
{"jsonrpc":"2.0","id":"1","result":{"status":"supported","reason":"exact synthetic demo profile match"}}
```

## Manifest and discovery

The API scans direct child directories of `ADAPTERS_DIR` (default `adapters`). Each adapter directory contains a `manifest.json` and its built executable. The executable path must be relative to that directory, resolve to an executable regular file, and stay within the directory. The manager validates the manifest schema, ID, semantic version, API version, and declared capabilities. Invalid entries are logged and skipped; duplicate adapter IDs are rejected.

Example manifest:

```json
{
  "manifest_version": 1,
  "api_version": "v1",
  "id": "adapter-android-demo",
  "name": "CARVION Synthetic Android Demo",
  "version": "0.1.0",
  "executable": "bin/adapter-android-demo",
  "capabilities": {
    "detect": true,
    "check_compatibility": true,
    "inspect_resources": true
  }
}
```

## Data boundary

The manager loads only code already present in the configured trusted local catalog. It does not download plugins. The process receives a minimal environment and no `DATABASE_URL`, PostgreSQL connection, or device handle. `Detect` receives JSON contents and `InspectResources` receives document contents as RPC payloads; adapters receive no filesystem path to an input source. Data collection and any future hardware access remain core responsibilities.

The separate process and reduced environment do not form an operating-system sandbox. Only install adapter executables from a source trusted by the machine owner; an executable still has the operating-system permissions of the user running CARVION.

## Demo adapter

`adapter-android-demo` uses synthetic fixtures only. `Detect` parses the JSON observation supplied by the caller. `CheckCompatibility` supports only an exact synthetic profile match; a known build on a different hardware revision is `unsupported`, and an unknown identity or build is `unknown`. `InspectResources` parses caller-supplied Android `strings.xml` content and skips entries marked `translatable="false"`.

The demo does not claim support for any real manufacturer, vehicle, multimedia unit, or firmware.

## HTTP API

- `GET /api/v1/adapters` returns the registered public summaries and does not expose executable paths.
- `POST /api/v1/compatibility/check` accepts `adapter_id`, `identity`, `hardware`, and `firmware`; it returns a structured compatibility result or a structured HTTP error.

There are no administrative registration or installation endpoints. The compatibility request body is limited to 64 KiB, uses strict JSON decoding, and requires all identity, hardware, and firmware fields. Unknown vehicles and firmware are adapter results with status `unknown`, not server errors.
