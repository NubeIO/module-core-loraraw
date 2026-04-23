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
