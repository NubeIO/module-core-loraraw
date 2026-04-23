# ZipHydroTap Decoder – Fix Notes

## Symptom

ZipHydroTap frames were received, matched, and decrypted, but **no points
updated**. Logs ended with a benign "done" line:

```
handleSerialPayload: LoRaRAW decrypt ok, decodedLen=72, dispatching to LoRaRAW handler
handleSerialPayload: done for address=00C032AA model=ZipHydroTap
```

Worked in `v1.1.1`, broke on `v1.2.1`.

## Cause

`handleLoRaRAWDevice` now calls decoders as
`DecodeUplink(dataHex, payloadBytes, ...)` — the wrapper is stripped into
`payloadBytes`. `DecodeZHT` was still parsing `dataHex`, which starts
with the 4-byte address, so `getPayloadType` read an address byte (`0x00`)
as the type → `ErrorData` → silent no-op.

Rubix was migrated to the new convention; ZHT was missed.

## Fix

One file: [`ziphydrotap.go`](./ziphydrotap.go). `DecodeZHT` now reads
from `payloadBytes`:

```go
payloadType := TZHTPayloadType(payloadBytes[0])
innerBytes  := payloadBytes[1:] // version byte; sub-decoders start at index:=1

switch payloadType {
case StaticData: return staticPayloadDecoder(innerBytes, ...)
case WriteData:  return writePayloadDecoder(innerBytes, ...)
case PollData:   return pollPayloadDecoder(innerBytes, ...)
}
```

Safe because `staticPayloadDecoder` / `writePayloadDecoder` /
`pollPayloadDecoder` already start at `index := 1` and
`payloadBytes[1]` is the version byte — same layout as the pre-`v1.1.1`
`hex.DecodeString(data[2:])`.

## Not changed (reverted during triage)

- `IsLoRaRAW: false` for ZipHydroTap in `codecs/codecs.go` — it does use
  LoRaRAW framing; comment is misleading.
- Extra legacy-decrypt branch in `handleSerialPayload` — unnecessary.

## Verification

- `go build ./...` clean.
- `TestZHTPayload` (poll / static / write) in `pkg/app_decrytped_test.go`
  still passes.

---

# Follow-up: Unencrypted LoRaRAW support

## Symptom

With `enable_decryption=false`, every ZHT frame logged:

```
handleSerialPayload: taking legacy handler path (legacyDevice=false, enableDecryption=false)
level=error msg="invalid legacy payload length"
```

Example frame (49 bytes, starts with `00C03200` + opts/nonce/len header):

```
00C03200013728030101C50352000D021E01FFFFFFFFC53400000000000000000000000000000083001700000000005400
```

## Cause

`handleSerialPayload` only had two branches: encrypted LoRaRAW or legacy.
When decryption was disabled, LoRaRAW-framed devices (ZHT, Rubix, UART)
fell through to `handleLegacyDevice`, which ran `CheckLength` on the raw
hex and rejected it.

## Fix

Added a middle branch in [`pkg/app.go`](../../pkg/app.go) →
`handleSerialPayload`:

```go
} else if !legacyDevice && !m.config.EnableDecryption && devDesc.IsLoRaRAW {
    // Layout: [addr:4][opts:1][nonce:1][len:1][payload:len][rssi:1][snr:1]
    dataBytes, _ := hex.DecodeString(dataHex)
    // bounds-check header + payload + rssi/snr
    payload := utils.StripLoRaRAWPayload(dataBytes)
    devDesc.DecodeUplink(dataHex, payload, devDesc, device, ...)
}
```

No CMAC, no AES — just strip the LoRaRAW wrapper and hand the inner
payload to the same decoder used by the encrypted path.

## Verification

- New fixture `ZHT-Unencrypted` in
  [`pkg/app_decrytped_test.go`](../../pkg/app_decrytped_test.go) uses the
  raw 49-byte frame above and asserts all 23 poll fields
  (e.g. `temperature_ntc_boiling=96.5`, `usage_energy_kwh=1350.9`,
  `filter_info_usage_litres_internal=131`).
- All existing ZHT fixtures (encrypted and decrypted) still pass.


