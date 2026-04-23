### How to build

```bash
go build -o module-core-loraraw
```

### MQTT

When `mqtt_enable: true` (default), the module connects to the broker
configured via `mqtt_broker` / `mqtt_username` / `mqtt_password` and
publishes two kinds of messages under `mqtt_topic_prefix`
(default `module-core-loraraw`).

Status (LWT, retained):

| Topic                           | Payload                |
|---------------------------------|------------------------|
| `module-core-loraraw/status`    | `online` / `offline`   |

Data topics:

#### `module-core-loraraw/raw`

Every received uplink is republished as an uppercase hex string, before
decoding. Useful for debugging and for downstream tools that want to do
their own parsing.

Payload (string):

```
00C032AAB0138AB28B6E9A969E7E9CCCA2032EA05837EDF19D35014D38697EB48F591B05E27C93089C3B6A6AF567CA517EAB07A8D8FB11A772C7B1310ABA061D8C6E933163A5AD085228
```

#### `module-core-loraraw/value`

Published once per frame after decoding completes. The JSON envelope
carries the device identity plus all decoded point values (including
`rssi` and `snr`).

Payload (JSON):

```json
{
  "device_address_uuid": "00C032AA",
  "device_name": "ZHT",
  "payload": {
    "rssi": -82,
    "snr": 10.0,
    "rebooted": 0,
    "sleep_mode_status": 1,
    "temperature_ntc_boiling": 98.0,
    "temperature_ntc_chilled": 8.4,
    "usage_energy_kwh": 4598.6,
    "filter_info_usage_litres_internal": 700,
    "filter_info_usage_days_internal": 160
  }
}
```

Notes:
- `device_address_uuid` is the device's `address_uuid` as stored in the DB.
- `device_name` is the device's `name`.
- `payload` only contains points that decoded successfully for the
  current frame (poll / write / static payloads emit different fields).
- Publishes are QoS 0, not retained. If the broker is down the data flow
  is unaffected and a debug log records the skipped publish.


