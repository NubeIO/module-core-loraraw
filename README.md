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

##### ZipHydroTap: encrypted vs published (decrypted) frames

When `decryption.enable: true` and the source transmits encrypted
LoRaRAW frames, the module decrypts the frame, strips the AES padding
and the 4-byte CMAC, and publishes the pre-encryption wire form to
`/raw`. Downstream consumers therefore do **not** need the decryption
key.

Published frame layout (56 bytes / 112 hex chars for a ZHT v2 poll):

```
[addr:4][opts:1][nonce:1][len:1][payload:len][rssi:1][snr:1]
```

Example pairs (encrypted on the wire → published on `/raw`):

| # | Encrypted (148 hex, on air) | Published (112 hex, on `/raw`) |
|---|-----------------------------|--------------------------------|
| 1 | `00C032AAD568D5E1183030FC473E898AFCCDC4AF9E55906575C263C42196157A4EB3FEF46BDB97D7EE97D029107E71E00747387359A77FFD18409CEAB90838ED0DCCD421676EFB315529` | `00C032AA01802F030201D40342006C023001FFFFFFFFADB3000000000000000000000000000000BF02A10000000000000000000000005529` |
| 2 | `00C032AA6962CF5A2BC6292AB4166C92695C30E2C243A14D132418D0B993966AEDB25FDD91025201F40BF41BC2E9467E9EFF33B6589AB1E681A79D295EE5DB57EEC3C5176ACEC7B45527` | `00C032AA01812F030201D40342006C023001FFFFFFFFADB3000000000000000000000000000000BF02A10000000000000000000000005527` |

Field breakdown of the published frame
`00C032AA 01 80 2F 03 02 01 D4 03 42 00 6C 02 30 01 FFFFFFFF ADB3 … 55 29`:

| bytes | hex        | field                                       |
|------:|------------|---------------------------------------------|
| 0–3   | `00C032AA` | device address                              |
| 4     | `01`       | opts                                        |
| 5     | `80`       | nonce (increments per frame)                |
| 6     | `2F`       | inner length = 47                           |
| 7     | `03`       | ZHT payload type (`3` = PollData)           |
| 8     | `02`       | ZHT packet version (v2 poll, 47 B)          |
| 9–53  | …          | ZHT poll fields                             |
| 54    | `55`       | RSSI (signed, appended by radio)            |
| 55    | `29`       | SNR × 4 (appended by radio)                 |

When `decryption.enable: false` or the source already sends
unencrypted LoRaRAW, the frame is published to `/raw` as received
(still with RSSI/SNR appended).

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


