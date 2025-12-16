## Module Core LoRaRAW

LoRa RAW serial bridge for the Nube iO Rubix platform. The plugin listens on a
single configured serial port, decodes LoRa uplinks from supported devices, and
writes values/meta back to Rubix via the module marshaller. It runs as a
HashiCorp go-plugin service and exposes a small HTTP surface through
`lib-module-go`.

For developer guide see [README-DEV.md](README-DEV.md).

---

## Quick start

Prerequisites: Go 1.21+

```bash
go build -o module-core-loraraw
```

---

## Configuration

Config is YAML and validated in `pkg/config.go`. Defaults:

```yaml
re_iteration_time: 5s  # delay between serial reconnect attempts
log_level: ERROR       # ERROR | INFO | DEBUG
```

- `log_level` is uppercased and coerced to a valid logrus level.
- `re_iteration_time` controls how long the run loop waits before retrying a
  failed serial open.

---

## Runtime lifecycle

- `Init(dbHelper, moduleName)` sets up the router and GRPC marshaller.
- `Enable()` starts the serial listener (`run`) and reports warnings if no
  networks exist for this plugin.
- `Disable()` signals the listener to stop and waits one extra second to avoid
  races with restarts.
- `addNetwork` enforces a single network per plugin, forces transport to
  `serial`, defaults baud to `38400`, and immediately kicks off the run loop.

---

## Serial uplink handling

- `run` opens the configured serial port, surfaces network faults on errors, and
  reconnects after `re_iteration_time`.
- Each open port is scanned line-by-line; frames are sent to `handleSerialPayload`.
- `handleSerialPayload` validates payload length, resolves the device by LoRa
  `address_uuid`, and:
  - logs RSSI for unknown devices;
  - decodes the payload for the device model;
  - writes point values and meta tags; and
  - updates RSSI/SNR points and clears device fault flags.
- LoRa RAW devices strip the header/CMAC/RSSI/SNR wrapper before decoding their
  payload (see `utils.CheckLoRaRAWPayloadLength` and `utils.StripLoRaRAWPayload`).

For a visual overview, see [FLOW.mmd](FLOW.mmd).

---

## HTTP API (router)

Routes live in `pkg/router.go` and are invoked via `Module.CallModule` (the
plugin itself is not an HTTP server).

- Schemas:
  - `GET /api/networks/schema`
  - `GET /api/devices/schema`
  - `GET /api/points/schema`
- Networks:
  - `POST /api/networks`
  - `PATCH /api/networks/:uuid`
  - `DELETE /api/networks/:uuid`
- Devices:
  - `POST /api/devices` (use query `with_points=true` to seed points)
  - `PATCH /api/devices/:uuid`
  - `DELETE /api/devices/:uuid`
- Points:
  - `POST /api/points`
  - `PATCH /api/points/:uuid`
  - `PATCH /api/points/:uuid/write`
  - `DELETE /api/points/:uuid`

Bodies use Rubix `model`/`dto` types from
[nubeio-rubix-lib-models-go](https://github.com/NubeIO/nubeio-rubix-lib-models-go).

For full API reference
see [README-API.md](README-API.md).

---

## Data model & device support

- Networks (`schema/network.go`):
  - Serial only; requires `serial_port` and `serial_baud_rate`; only one network
    is allowed for this plugin.
  - `history_enable` is available and passed through to Rubix.
- Devices (`schema/device.go`):
  - Models: `THLM`, `THL`, `TH`, `MicroEdgeV1`, `MicroEdgeV2`, `ZipHydroTap`,
    `Rubix`.
  - `address_uuid` must be 8 hex characters and unique (uppercased on insert).
  - Updating a device re-applies the address to all existing points.
- Points (`schema/point.go`):
  - Created as `analog_input` with `io_type=RAW`, `Enable=true`,
    `EnableWriteable=false`, and read-only `WriteMode`.
  - Names are title-cased from the device-defined point names; RSSI/SNR points
    are always created.

Device-specific decoding:

- **MicroEdge V1/V2** – pulse count, battery voltage, `ai_1..ai_3` plus RSSI/SNR;
  applies type-specific conversion when `io_type` is not RAW.
- **Droplet TH/THL/THLM** – temperature, pressure, humidity, voltage and
  optionally light/motion depending on model.
- **ZipHydroTap (LoRa RAW)** – parses static/write/poll payloads; populates a
  large set of operating points (timers, faults, usage) and meta tags such as
  firmware/build versions and derived Modbus address.
- **Rubix (LoRa RAW)** – flexible bit-packed payload; generates positional point
  names per metadata header (temperature, RH, lux, digital, analog, counters,
  firmware/hardware versions, typed integers/floats, etc.).

---

## Logging and diagnostics

- Log level comes from config and is applied via `logger.SetLogger`.
- Serial open failures set a network fault with the error message; successful
  opens clear it.
- Successful payloads clear device faults; unknown devices log their ID and RSSI
  to help with onboarding.

---

## Directory map

- `main.go` — HashiCorp go-plugin entrypoint.
- `pkg/` — module lifecycle, config, router handlers, serial loop, point/meta
  updates.
- `decoder/` — device descriptions and payload decoding logic.
- `schema/` — JSON schemas for network, device, and point forms.
- `utils/` — LoRa RAW helpers and reflection utilities.
- `logger/` — logrus setup.

---