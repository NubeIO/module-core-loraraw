# Module Core LoRaRAW – HTTP API Reference

Routes are registered in `pkg/router.go` and are invoked by the host via
`Module.CallModule` (the plugin itself does not run an HTTP server). Bodies use
Rubix `model`/`dto` types from
[`nubeio-rubix-lib-models-go`](https://github.com/NubeIO/nubeio-rubix-lib-models-go).

## Base
- Content type: `application/json`
- Success status: typically `200 OK`; creation also returns `200 OK` (host does
  not use 201). Errors bubble as 4xx/5xx from the host.

## Schemas
- `GET /api/networks/schema` – JSON schema for `Network`.
- `GET /api/devices/schema` – JSON schema for `Device`.
- `GET /api/points/schema` – JSON schema for `Point`.

## Networks
- `POST /api/networks` – create network (serial only).
  - Body: `model.Network` with `serial_port` and `serial_baud_rate` set; plugin
    name is inferred.
  - Only one network for this plugin is allowed; a second will be rejected.
  - Success: `200 OK` with created `Network` including UUID.
- `PATCH /api/networks/:uuid` – update network.
  - Body: partial/complete `model.Network` (e.g., change serial port).
  - Success: `200 OK` with updated `Network`.
- `DELETE /api/networks/:uuid` – delete network.
  - Success: `200 OK` with boolean.

## Devices
- `POST /api/devices` – create device.
  - Body: `model.Device` with `network_uuid`, `model` (one of `THLM`, `THL`,
    `TH`, `MicroEdgeV1`, `MicroEdgeV2`, `ZipHydroTap`, `Rubix`), and
    `address_uuid` (8 hex chars, unique, uppercased internally).
  - Query: `with_points=true` seeds default points for the device model.
  - Success: `200 OK` with created `Device`.
- `PATCH /api/devices/:uuid` – update device.
  - Body: partial/complete `model.Device`. Changing `address_uuid` updates all
    associated points.
  - Success: `200 OK` with updated `Device`.
- `DELETE /api/devices/:uuid` – delete device.
  - Success: `200 OK` with boolean.

## Points
- `POST /api/points` – create point.
  - Body: `model.Point` (device uuid, name/description, enable).
  - Points created by the plugin default to `analog_input`, `io_type=RAW`,
    `write_mode=ReadOnly`, `enable_writeable=false`.
  - Success: `200 OK` with created `Point`.
- `PATCH /api/points/:uuid` – update point.
  - Body: partial/complete `model.Point`.
  - Success: `200 OK` with updated `Point`.
- `DELETE /api/points/:uuid` – delete point.
  - Success: `200 OK` with boolean.
- `PATCH /api/points/:uuid/write` – queue a write (rarely used for LoRa sensors,
  but supported for writable models).
  - Body: `dto.PointWriter` (priority array/present value).
  - Success: `200 OK` with updated `Point` after write request is queued.

## Examples (host forwards to `CallModule`)
- Create network:
```bash
curl -X POST http://<host>/api/networks \
  -H "Content-Type: application/json" \
  -d '{"name":"loranet","serial_port":"/dev/ttyUSB0","serial_baud_rate":38400,"enable":true}'
```

- Create device (and seed points):
```bash
curl -X POST "http://<host>/api/devices?with_points=true" \
  -H "Content-Type: application/json" \
  -d '{"network_uuid":"<net>","address_uuid":"A1B2C3D4","model":"THLM","enable":true}'
```

- Write to a point (if the model supports writes):
```bash
curl -X PATCH http://<host>/api/points/<point_uuid>/write \
  -H "Content-Type: application/json" \
  -d '{"priority":{"_16":12.3}}'
```
