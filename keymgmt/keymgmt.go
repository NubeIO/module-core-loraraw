// Package keymgmt decides which AES key Rubix CE uses for a LoRa device, and
// what that implies for how strictly its frames are checked.
//
// Design (Phương án B - "nhập khoá hai phía"): the key is produced OUTSIDE this
// system - at manufacturing, or by whoever commissions the device - and simply
// typed into both ends: the device (AT+AES= on STM32, param_set 0x0220 on
// ESP32) and the "Device Key" box in Rubix CE. There is no provisioning API and
// no key generation here; CE only needs to know the key and enforce it.
//
// The package deliberately owns ONLY key selection. It performs no crypto (see
// aesutils), no transport (see pkg/serial.go) and no device lookup (see
// pkg/app.go) - the separation required by doc 10 section 10.1.
package keymgmt

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
)

// KeyLen is the AES-128 key length in bytes. The wire protocol and every device
// firmware are fixed at AES-128 (doc 03).
const KeyLen = 16

// MetaKeyMode optionally pins a device's mode on the framework Device record.
// It is only needed to mark a device as explicitly UNENCRYPTED; otherwise the
// mode is inferred from whether a per-device key is set.
//
// DeviceMetaTag is a key/value store the framework already supports, so this
// needs no model change and no migration.
const MetaKeyMode = "lora_key_mode"

// Mode is how a device's traffic is protected on the wire.
type Mode string

const (
	// ModeUnencrypted: device sends plaintext. Explicit per-device opt-in
	// only, never a fallback (doc 09 section 9.3).
	ModeUnencrypted Mode = "UNENCRYPTED"
	// ModeSharedKey: device uses the fleet-wide default key - the status quo
	// for anything not yet given its own key.
	ModeSharedKey Mode = "SHARED_KEY"
	// ModePerDeviceKey: device has its own key. The target state.
	ModePerDeviceKey Mode = "PER_DEVICE_KEY"
)

// ParseMode converts stored text to a Mode. ok is false for unknown values so
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

// Key is raw AES key material, with a redacting String so it cannot be printed
// by accident (doc 07 section 7.5).
type Key []byte

// String never reveals key material.
func (k Key) String() string { return fmt.Sprintf("[REDACTED %d-byte key]", len(k)) }

// Zero overwrites the key material in place.
func (k Key) Zero() {
	for i := range k {
		k[i] = 0
	}
}

// Resolution is what the caller needs to handle one device's frame.
type Resolution struct {
	Key  Key
	Mode Mode
	// UsesSharedKey reports that this device's key is the fleet-wide default
	// - either because none was set, or because the shared key itself was
	// pasted into the Device Key box. The second case looks like provisioning
	// but protects nothing, so callers should warn about it.
	UsesSharedKey bool
}

// EncryptionRequired reports whether a plaintext frame from this device must be
// rejected. Only a genuine per-device key demands it: that is the guard which
// closes the downgrade hole (G3).
func (r Resolution) EncryptionRequired() bool { return r.Mode == ModePerDeviceKey }

// Zero wipes the resolved key material.
func (r Resolution) Zero() { r.Key.Zero() }

// Resolver turns a Device record into a Resolution. Stateless, safe for
// concurrent use.
type Resolver struct {
	defaultKeyHex string
}

// NewResolver builds a Resolver over the module's configured default key.
func NewResolver(defaultKeyHex string) *Resolver {
	return &Resolver{defaultKeyHex: defaultKeyHex}
}

var errNilDevice = errors.New("keymgmt: nil device")

// Resolve selects the key for a device and reports its mode.
//
// Key selection matches the previous behaviour exactly: Device.Manufacture when
// set, otherwise the module's default key.
//
// The mode is inferred - a device with its own key is PER_DEVICE_KEY, anything
// else is SHARED_KEY - unless a meta tag pins it (the only real use being an
// explicit UNENCRYPTED opt-in). An unparseable tag falls back to the inferred
// value so corrupt storage can never silently downgrade a device.
func (r *Resolver) Resolve(device *model.Device) (Resolution, error) {
	if device == nil {
		return Resolution{}, errNilDevice
	}

	defaultKey, err := decodeKey(r.defaultKeyHex, "default key")
	if err != nil {
		return Resolution{}, err
	}

	key := defaultKey
	if device.Manufacture != "" {
		key, err = decodeKey(device.Manufacture, "per-device key")
		if err != nil {
			return Resolution{}, err
		}
	}

	// Pasting the shared key into the Device Key box is not provisioning:
	// treat it as SHARED_KEY so it is not mistaken for a protected device.
	usesShared := equalKeys(key, defaultKey)

	mode := ModePerDeviceKey
	if usesShared {
		mode = ModeSharedKey
	}
	if pinned, ok := ParseMode(metaTagValue(device, MetaKeyMode)); ok {
		mode = pinned
	}

	return Resolution{Key: key, Mode: mode, UsesSharedKey: usesShared}, nil
}

// decodeKey parses a 16-byte hex key. Dash separators are accepted because the
// ESP32 console prints keys that way (AA-BB-CC-...), and an operator copying
// from `param_get 0x0220` should not have to strip them by hand.
func decodeKey(hexKey, which string) (Key, error) {
	cleaned := strings.ReplaceAll(strings.TrimSpace(hexKey), "-", "")
	cleaned = strings.ReplaceAll(cleaned, " ", "")
	raw, err := hex.DecodeString(cleaned)
	if err != nil {
		// Never echo the value: it is secret material.
		return nil, fmt.Errorf("keymgmt: %s is not valid hex: %w", which, err)
	}
	if len(raw) != KeyLen {
		return nil, fmt.Errorf("keymgmt: %s must be %d bytes, got %d", which, KeyLen, len(raw))
	}
	return Key(raw), nil
}

func equalKeys(a, b Key) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func metaTagValue(device *model.Device, key string) string {
	for _, t := range device.MetaTags {
		if t != nil && t.Key == key {
			return t.Value
		}
	}
	return ""
}
