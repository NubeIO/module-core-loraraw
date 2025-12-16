# Module Core LoRaRAW – Developer Guide

Notes for contributors working on the LoRa RAW serial bridge plugin for the
Rubix platform. Covers structure, build/test, runtime behavior, and host
integration.

## Architecture at a glance
- Entrypoint `main.go` starts a HashiCorp go-plugin server (`nmodule.NubeModule`)
  via `lib-module-go`.
- Routes are registered in `pkg/router.go` and invoked through
  `Module.CallModule`; there is no standalone HTTP server.
- Serial loop lives in `pkg/run.go`/`pkg/serial.go`: opens the single configured
  serial port, scans lines, dispatches to `handleSerialPayload`.
- Device payload decoding is in `decoder/` with per-model handlers for
  MicroEdge, Droplet, ZipHydroTap, and Rubix.
- Schemas for network/device/point are in `schema/`.
- Logging setup is in `logger/`; helpers in `utils/`.

## Prerequisites
- Go 1.21+
- Docker (optional) if you use `build.bash` or the cross-build Dockerfiles.
- A Rubix host (or `lib-module-go` harness) to load the plugin; it is not a
  standalone HTTP server.

## Project layout (quick map)
- `main.go` – go-plugin wiring.
- `pkg/` – module lifecycle, config, router handlers, serial loop, point/meta
  updates.
- `decoder/` – device descriptions and payload decoders.
- `schema/` – JSON schemas for network/device/point.
- `utils/` – LoRa RAW helpers (strip/length checks) and reflection helpers.
- `logger/` – logrus setup.

## Build & run
Local build (amd64):
```bash
go build -o module-core-loraraw
```

You can run `go run .` for quick smoke testing; it will wait idle until a host
connects over the go-plugin protocol and passes config.

### Cross-compiling with Docker
- `build.bash` wraps `Dockerfile.module` to build and copy out `module-core-loraraw`.
- `Dockerfile` builds amd64/armv7 binaries using Go 1.21.13.

## Configuration
Validated in `pkg/config.go`. Defaults:
```yaml
re_iteration_time: 5s
log_level: ERROR   # coerced to a valid logrus level and uppercased
```
Notes:
- Config is supplied by the host via `ValidateAndSetConfig`; there is no local
  file loader.
- `re_iteration_time` governs retry delay after serial open failures.

## HTTP API surface (via `CallModule`)
Registered in `pkg/router.go` using Rubix `model`/`dto` bodies:
- Schemas: `GET /api/networks/schema`, `/api/devices/schema`, `/api/points/schema`
- CRUD: `POST/PATCH/DELETE /api/{networks,devices,points}`
- Point write: `PATCH /api/points/:uuid/write` (rare for LoRa but supported)

Example create-device (host forwards to `CallModule`):
```bash
curl -X POST http://<host>/api/devices?with_points=true \
  -H "Content-Type: application/json" \
  -d '{"network_uuid":"<net>","address_uuid":"A1B2C3D4","model":"THLM","enable":true}'
```

## Runtime model (developer notes)
- Enable:
  - `Enable()` fetches networks by plugin name; warns if none exist.
  - Starts the serial run loop (`run`) with an interrupt channel for disable.
- Serial loop:
  - `SerialOpen` enforces exactly one network and requires `serial_port` and
    `serial_baud_rate`; sets/clears network fault on failure/success.
  - A scanner reads lines from the port and feeds `handleSerialPayload`; errors
    trigger reconnect after `re_iteration_time`.
- Payload handling:
  - Rejects short frames; resolves device by `address_uuid` (with points + meta).
  - Unknown devices: log ID + RSSI only.
  - LoRa RAW devices strip header/CMAC/RSSI/SNR before decoding.
  - Per-model decoder updates/creates points, meta tags, RSSI/SNR, and clears
    device faults.
- Point creation/update:
  - Device add (with `with_points=true`) seeds model-specific points plus
    RSSI/SNR; all are `analog_input`, `io_type=RAW`, `write_mode=ReadOnly`,
    `EnableWriteable=false`.
  - Changing a device `address_uuid` rewrites it onto all points.

## Logging & debugging
- `log_level` applied via `logger.SetLogger`; uses logrus text formatter.
- Serial open failures set a network fault; successes clear it.
- Payload errors log the decode failure; unknown devices log at info with RSSI
  to aid commissioning.
- See `FLOW.mmd` for a LoRaRAW run/handling diagram.

## Tests and checks
- Go tests: `go test ./...` (coverage is minimal).
- Static checks (manual): `go vet ./...`.

## Developing against Rubix
- CRUD, point writes, and meta updates flow through `nmodule.GRPCMarshaller`.
- Typical workflow on a Rubix host:
  - Build and place the binary where the host expects modules
    (e.g. `/data/rubix-os/data/modules/module-core-loraraw/<version>/module-core-loraraw`).
  - Ensure the host passes YAML config (port/baud, log level) on enable.
  - Add a network, then devices with `with_points=true`, and watch point updates
    as payloads arrive.

## Release tips
- No version file; bump artifacts manually as needed.
- Use `build.bash` or `Dockerfile` to produce amd64/armv7 binaries.
- Smoke test on a Rubix host: enable module, create network + device, send a
  sample payload, confirm points/meta update and faults stay clear.
