// Package keymgmt holds the LoRa device key model: what key a device uses,
// which encryption mode it is in, and where it sits in the provisioning
// lifecycle.
//
// It deliberately owns ONLY key selection and lifecycle state. It performs no
// crypto (see aesutils), no transport (see pkg/serial.go) and no device lookup
// (see pkg/app.go getDeviceByLoRaAddress) — the separation of concerns required
// by the design docs (doc 10 section 10.1).
//
// Step S1 scope: introduce the vocabulary and the resolver WITHOUT changing
// runtime behaviour. Key selection here is byte-identical to the previous
// Module.getEncryptionKey: Device.Manufacture when set, otherwise the module's
// default key. Mode and ProvisionState are recorded/derived but nothing acts on
// them yet — enforcement (encryption_required) lands in S4.
package keymgmt

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
)

// KeyLen is the AES-128 key length in bytes. The wire protocol and every
// device firmware are fixed at AES-128 (see doc 03).
const KeyLen = 16

// Meta-tag keys used to persist per-device key state on the framework Device
// record. DeviceMetaTag is a plain key/value store already supported by the
// framework, so storing state here needs no model change and no migration.
const (
	MetaKeyMode           = "lora_key_mode"
	MetaKeyProvisionState = "lora_provision_state"
)

// Mode is how a device's traffic is protected on the wire.
type Mode string

const (
	// ModeUnencrypted: device sends plaintext frames. Must be an explicit
	// per-device opt-in, never a fallback (doc 09 section 9.3).
	ModeUnencrypted Mode = "UNENCRYPTED"
	// ModeSharedKey: device uses the fleet-wide default key. The status quo
	// for every un-provisioned device, and the only option for legacy
	// whole-frame-encrypted devices.
	ModeSharedKey Mode = "SHARED_KEY"
	// ModePerDeviceKey: device has its own key. The target state.
	ModePerDeviceKey Mode = "PER_DEVICE_KEY"
)

// ProvisionState is where the device sits in the key lifecycle (doc 08 section 8.1).
type ProvisionState string

const (
	StateUnprovisioned ProvisionState = "UNPROVISIONED"
	StatePending       ProvisionState = "PENDING"
	StateActive        ProvisionState = "ACTIVE"
	StateRetired       ProvisionState = "RETIRED"
)

// ParseMode converts stored text to a Mode. ok is false for unknown values, so
// callers can fall back rather than trust unvalidated storage.
func ParseMode(s string) (Mode, bool) {
	switch Mode(strings.ToUpper(strings.TrimSpace(s))) {
	case ModeUnencrypted:
		return ModeUnencrypted, true
	case ModeSharedKey:
		return ModeSharedKey, true
	case ModePerDeviceKey:
		return ModePerDeviceKey, true
	default:
		return "", false
	}
}

// ParseProvisionState converts stored text to a ProvisionState.
func ParseProvisionState(s string) (ProvisionState, bool) {
	switch ProvisionState(strings.ToUpper(strings.TrimSpace(s))) {
	case StateUnprovisioned:
		return StateUnprovisioned, true
	case StatePending:
		return StatePending, true
	case StateActive:
		return StateActive, true
	case StateRetired:
		return StateRetired, true
	default:
		return "", false
	}
}

// Key is raw AES key material. It carries a redacting String so it can never be
// printed by accident (doc 07 section 7.5).
type Key []byte

// String never reveals key material.
func (k Key) String() string { return fmt.Sprintf("[REDACTED %d-byte key]", len(k)) }

// Zero overwrites the key material in place. Call it once the crypto operation
// is done to keep raw key bytes out of memory longer than necessary.
func (k Key) Zero() {
	for i := range k {
		k[i] = 0
	}
}

// Resolution is everything the caller needs to handle one device's frame.
type Resolution struct {
	// Key is the device's live key — what it is expected to be using now.
	Key Key
	// PendingKey is a key staged by an in-flight provisioning/rotation, or
	// nil. During that window the device may already have switched to it, so
	// a frame that fails under Key may still be genuine under PendingKey.
	// Trying it is NOT a downgrade: both keys belong to this device, and the
	// window is closed as soon as the new key is confirmed (doc 08 section 8.2).
	PendingKey Key
	Mode       Mode
	State      ProvisionState
}

// EncryptionRequired reports whether a plaintext frame from this device must be
// rejected. Only a per-device-key device demands it: that is the guard which
// closes the downgrade hole (G3) once S4 enforces it.
func (r Resolution) EncryptionRequired() bool { return r.Mode == ModePerDeviceKey }

// Zero wipes every key this Resolution holds.
func (r Resolution) Zero() {
	r.Key.Zero()
	r.PendingKey.Zero()
}

// Resolver turns a Device record into a Resolution. It is stateless and safe
// for concurrent use.
type Resolver struct {
	defaultKeyHex string
}

// NewResolver builds a Resolver over the module's configured default key.
func NewResolver(defaultKeyHex string) *Resolver {
	return &Resolver{defaultKeyHex: defaultKeyHex}
}

var errNilDevice = errors.New("keymgmt: nil device")

// Resolve selects the key for a device and reports its mode and lifecycle
// state.
//
// Key selection is intentionally identical to the pre-S1 behaviour: the
// per-device key from Device.Manufacture when present, otherwise the shared
// default key.
//
// Mode and State come from meta tags when they hold a valid value; otherwise
// they are inferred from whether a per-device key is set. Unparseable stored
// values fall back to the inferred value rather than failing the frame, so a
// corrupted tag can never silently downgrade a device.
func (r *Resolver) Resolve(device *model.Device) (Resolution, error) {
	if device == nil {
		return Resolution{}, errNilDevice
	}

	hexKey := r.defaultKeyHex
	perDevice := device.Manufacture != ""
	if perDevice {
		hexKey = device.Manufacture
	}

	key, err := decodeKey(hexKey, perDevice)
	if err != nil {
		return Resolution{}, err
	}

	return Resolution{
		Key:        key,
		PendingKey: resolvePendingKey(device),
		Mode:       resolveMode(device, perDevice),
		State:      resolveState(device, perDevice),
	}, nil
}

func decodeKey(hexKey string, perDevice bool) (Key, error) {
	which := "default key"
	if perDevice {
		which = "per-device key"
	}
	raw, err := hex.DecodeString(strings.TrimSpace(hexKey))
	if err != nil {
		// Never echo the value itself — it is secret material.
		return nil, fmt.Errorf("keymgmt: %s is not valid hex: %w", which, err)
	}
	if len(raw) != KeyLen {
		return nil, fmt.Errorf("keymgmt: %s must be %d bytes, got %d", which, KeyLen, len(raw))
	}
	return Key(raw), nil
}

// resolvePendingKey returns the staged key, or nil when there is none. A
// corrupt stored value is ignored rather than fatal: a bad pending key must
// never take down a device whose live key is fine.
func resolvePendingKey(device *model.Device) Key {
	raw := metaTagValue(device, MetaKeyPendingKey)
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	k, err := decodeKey(raw, true)
	if err != nil {
		return nil
	}
	return k
}

func resolveMode(device *model.Device, perDevice bool) Mode {
	if v, ok := ParseMode(metaTagValue(device, MetaKeyMode)); ok {
		return v
	}
	if perDevice {
		return ModePerDeviceKey
	}
	return ModeSharedKey
}

func resolveState(device *model.Device, perDevice bool) ProvisionState {
	if v, ok := ParseProvisionState(metaTagValue(device, MetaKeyProvisionState)); ok {
		return v
	}
	if perDevice {
		return StateActive
	}
	return StateUnprovisioned
}

func metaTagValue(device *model.Device, key string) string {
	for _, t := range device.MetaTags {
		if t != nil && t.Key == key {
			return t.Value
		}
	}
	return ""
}
