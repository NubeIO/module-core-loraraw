package keymgmt

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
)

// Clearly-fake test vectors - never real production keys (doc 07 section 7.5).
const (
	fakeDefaultKeyHex   = "00112233445566778899AABBCCDDEEFF"
	fakePerDeviceKeyHex = "0F1E2D3C4B5A69788796A5B4C3D2E1F0"
)

func newDevice(manufacture string, metaTags ...*model.DeviceMetaTag) *model.Device {
	addr := "5CC0D947"
	d := &model.Device{}
	d.AddressUUID = &addr // from the embedded CommonDevice
	d.Manufacture = manufacture
	d.MetaTags = metaTags
	return d
}

func metaTag(key, value string) *model.DeviceMetaTag {
	return &model.DeviceMetaTag{Key: key, Value: value}
}

// --- Key selection: identical to the pre-change getEncryptionKey ---

func TestResolve_EmptyManufactureUsesDefaultKey(t *testing.T) {
	got, err := NewResolver(fakeDefaultKeyHex).Resolve(newDevice(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want, _ := hex.DecodeString(fakeDefaultKeyHex)
	if !bytes.Equal(got.Key, want) {
		t.Errorf("key = %X, want %X", got.Key, want)
	}
	if got.Mode != ModeSharedKey {
		t.Errorf("mode = %q, want %q", got.Mode, ModeSharedKey)
	}
}

func TestResolve_ManufactureGivesPerDeviceKey(t *testing.T) {
	got, err := NewResolver(fakeDefaultKeyHex).Resolve(newDevice(fakePerDeviceKeyHex))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want, _ := hex.DecodeString(fakePerDeviceKeyHex)
	if !bytes.Equal(got.Key, want) {
		t.Errorf("key = %X, want %X", got.Key, want)
	}
	if got.Mode != ModePerDeviceKey {
		t.Errorf("mode = %q, want %q", got.Mode, ModePerDeviceKey)
	}
}

func TestResolve_AcceptsLowercaseAndSpaces(t *testing.T) {
	in := "  " + strings.ToLower(fakePerDeviceKeyHex) + "  "
	got, err := NewResolver(fakeDefaultKeyHex).Resolve(newDevice(in))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want, _ := hex.DecodeString(fakePerDeviceKeyHex)
	if !bytes.Equal(got.Key, want) {
		t.Errorf("key = %X, want %X", got.Key, want)
	}
}

// A dash-separated key (the format the ESP32 console uses) must be accepted
// too: an operator copying the key out of `param_get 0x0220` gets dashes, and
// pasting that into the UI should just work rather than fail cryptically.
func TestResolve_AcceptsDashSeparatedKey(t *testing.T) {
	dashed := "0F-1E-2D-3C-4B-5A-69-78-87-96-A5-B4-C3-D2-E1-F0"
	got, err := NewResolver(fakeDefaultKeyHex).Resolve(newDevice(dashed))
	if err != nil {
		t.Fatalf("dash-separated key must be accepted, got: %v", err)
	}
	want, _ := hex.DecodeString(fakePerDeviceKeyHex)
	if !bytes.Equal(got.Key, want) {
		t.Errorf("key = %X, want %X", got.Key, want)
	}
}

// --- Error paths ---

func TestResolve_NilDeviceIsError(t *testing.T) {
	if _, err := NewResolver(fakeDefaultKeyHex).Resolve(nil); err == nil {
		t.Fatal("expected error for nil device")
	}
}

func TestResolve_InvalidHexIsError(t *testing.T) {
	_, err := NewResolver(fakeDefaultKeyHex).Resolve(newDevice("ZZZZ2233445566778899AABBCCDDEEFF"))
	if err == nil {
		t.Fatal("expected error for non-hex key")
	}
}

func TestResolve_WrongLengthIsError(t *testing.T) {
	if _, err := NewResolver(fakeDefaultKeyHex).Resolve(newDevice("0011223344556677")); err == nil {
		t.Fatal("expected error for 8-byte key")
	}
}

func TestResolve_InvalidDefaultKeyIsError(t *testing.T) {
	if _, err := NewResolver("not-hex").Resolve(newDevice("")); err == nil {
		t.Fatal("expected error for invalid default key")
	}
}

// --- The shared-key-in-the-Device-Key-box trap ---

// Typing the fleet-wide shared key into the per-device Device Key box looks
// like provisioning but protects nothing. The device must still be reported as
// SHARED_KEY so it is not mistaken for a provisioned one.
func TestResolve_SharedKeyPastedAsDeviceKeyIsNotPerDevice(t *testing.T) {
	got, err := NewResolver(fakeDefaultKeyHex).Resolve(newDevice(fakeDefaultKeyHex))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Mode != ModeSharedKey {
		t.Errorf("mode = %q, want %q - pasting the shared key is NOT per-device provisioning",
			got.Mode, ModeSharedKey)
	}
	if !got.UsesSharedKey {
		t.Error("UsesSharedKey must be true so the caller can warn the operator")
	}
	if got.EncryptionRequired() {
		t.Error("a device on the shared key must not be treated as encryption-required")
	}
}

func TestResolve_RealPerDeviceKeyIsNotFlaggedShared(t *testing.T) {
	got, _ := NewResolver(fakeDefaultKeyHex).Resolve(newDevice(fakePerDeviceKeyHex))
	if got.UsesSharedKey {
		t.Error("a genuinely different key must not be flagged as shared")
	}
}

// --- Explicit unencrypted opt-in ---

func TestResolve_MetaTagCanMarkDeviceUnencrypted(t *testing.T) {
	d := newDevice("", metaTag(MetaKeyMode, string(ModeUnencrypted)))
	got, err := NewResolver(fakeDefaultKeyHex).Resolve(d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Mode != ModeUnencrypted {
		t.Errorf("mode = %q, want %q", got.Mode, ModeUnencrypted)
	}
}

func TestResolve_GibberishMetaTagFallsBackSafely(t *testing.T) {
	d := newDevice(fakePerDeviceKeyHex, metaTag(MetaKeyMode, "NONSENSE"))
	got, err := NewResolver(fakeDefaultKeyHex).Resolve(d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Must not silently downgrade: falls back to the inferred mode.
	if got.Mode != ModePerDeviceKey {
		t.Errorf("mode = %q, want fallback %q", got.Mode, ModePerDeviceKey)
	}
}

// --- EncryptionRequired: the downgrade guard ---

func TestEncryptionRequired(t *testing.T) {
	for _, c := range []struct {
		mode Mode
		want bool
	}{
		{ModePerDeviceKey, true},
		{ModeSharedKey, false},
		{ModeUnencrypted, false},
	} {
		if got := (Resolution{Mode: c.mode}).EncryptionRequired(); got != c.want {
			t.Errorf("mode %q: EncryptionRequired() = %v, want %v", c.mode, got, c.want)
		}
	}
}

// --- Key hygiene ---

func TestKey_StringIsRedacted(t *testing.T) {
	k := Key(mustHex(fakePerDeviceKeyHex))
	if s := k.String(); strings.Contains(strings.ToUpper(s), "0F1E2D") || s == "" {
		t.Errorf("Key.String() must redact and be non-empty, got %q", s)
	}
}

func TestKey_ZeroWipes(t *testing.T) {
	k := Key(mustHex(fakePerDeviceKeyHex))
	k.Zero()
	for i, b := range k {
		if b != 0 {
			t.Fatalf("byte %d not zeroed: %#x", i, b)
		}
	}
}

func TestParseMode(t *testing.T) {
	for in, want := range map[string]struct {
		m  Mode
		ok bool
	}{
		"PER_DEVICE_KEY": {ModePerDeviceKey, true},
		"shared_key":     {ModeSharedKey, true},
		"UNENCRYPTED":    {ModeUnencrypted, true},
		"":               {"", false},
		"NONSENSE":       {"", false},
	} {
		got, ok := ParseMode(in)
		if ok != want.ok || (ok && got != want.m) {
			t.Errorf("ParseMode(%q) = (%q,%v), want (%q,%v)", in, got, ok, want.m, want.ok)
		}
	}
}

func mustHex(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}
