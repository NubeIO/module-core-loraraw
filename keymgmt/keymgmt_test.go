package keymgmt

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
)

// Clearly-fake test vectors. Never real production keys (see doc 07 section 7.5).
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

// --- Resolve: key selection must stay byte-identical to the old getEncryptionKey ---

func TestResolve_EmptyManufactureUsesDefaultKey(t *testing.T) {
	r := NewResolver(fakeDefaultKeyHex)
	got, err := r.Resolve(newDevice(""))
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
	if got.State != StateUnprovisioned {
		t.Errorf("state = %q, want %q", got.State, StateUnprovisioned)
	}
}

func TestResolve_ManufactureOverridesDefaultKey(t *testing.T) {
	r := NewResolver(fakeDefaultKeyHex)
	got, err := r.Resolve(newDevice(fakePerDeviceKeyHex))
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
	if got.State != StateActive {
		t.Errorf("state = %q, want %q", got.State, StateActive)
	}
}

func TestResolve_LowercaseManufactureIsAccepted(t *testing.T) {
	r := NewResolver(fakeDefaultKeyHex)
	got, err := r.Resolve(newDevice(strings.ToLower(fakePerDeviceKeyHex)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want, _ := hex.DecodeString(fakePerDeviceKeyHex)
	if !bytes.Equal(got.Key, want) {
		t.Errorf("key = %X, want %X", got.Key, want)
	}
}

// --- Error paths ---

func TestResolve_NilDeviceIsError(t *testing.T) {
	r := NewResolver(fakeDefaultKeyHex)
	if _, err := r.Resolve(nil); err == nil {
		t.Fatal("expected error for nil device, got nil")
	}
}

func TestResolve_InvalidManufactureHexIsError(t *testing.T) {
	r := NewResolver(fakeDefaultKeyHex)
	_, err := r.Resolve(newDevice("ZZZZ2233445566778899AABBCCDDEEFF"))
	if err == nil {
		t.Fatal("expected error for non-hex manufacture, got nil")
	}
}

func TestResolve_WrongKeyLengthIsError(t *testing.T) {
	r := NewResolver(fakeDefaultKeyHex)
	// 8 bytes instead of 16.
	if _, err := r.Resolve(newDevice("0011223344556677")); err == nil {
		t.Fatal("expected error for short key, got nil")
	}
}

func TestResolve_InvalidDefaultKeyIsError(t *testing.T) {
	r := NewResolver("not-hex")
	if _, err := r.Resolve(newDevice("")); err == nil {
		t.Fatal("expected error for invalid default key, got nil")
	}
}

// --- Meta tags override the inferred mode/state ---

func TestResolve_MetaTagOverridesMode(t *testing.T) {
	r := NewResolver(fakeDefaultKeyHex)
	// Manufacture is empty (would infer SHARED_KEY) but the device is
	// explicitly marked UNENCRYPTED.
	d := newDevice("", metaTag(MetaKeyMode, string(ModeUnencrypted)))
	got, err := r.Resolve(d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Mode != ModeUnencrypted {
		t.Errorf("mode = %q, want %q", got.Mode, ModeUnencrypted)
	}
}

func TestResolve_MetaTagOverridesState(t *testing.T) {
	r := NewResolver(fakeDefaultKeyHex)
	d := newDevice(fakePerDeviceKeyHex, metaTag(MetaKeyProvisionState, string(StatePending)))
	got, err := r.Resolve(d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.State != StatePending {
		t.Errorf("state = %q, want %q", got.State, StatePending)
	}
}

func TestResolve_UnknownMetaTagValueFallsBackToInference(t *testing.T) {
	r := NewResolver(fakeDefaultKeyHex)
	d := newDevice(fakePerDeviceKeyHex, metaTag(MetaKeyMode, "GIBBERISH"))
	got, err := r.Resolve(d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Must not crash and must not silently downgrade: falls back to the
	// value inferred from Manufacture.
	if got.Mode != ModePerDeviceKey {
		t.Errorf("mode = %q, want fallback %q", got.Mode, ModePerDeviceKey)
	}
}

func TestResolve_MetaTagCaseInsensitive(t *testing.T) {
	r := NewResolver(fakeDefaultKeyHex)
	d := newDevice("", metaTag(MetaKeyMode, "unencrypted"))
	got, err := r.Resolve(d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Mode != ModeUnencrypted {
		t.Errorf("mode = %q, want %q", got.Mode, ModeUnencrypted)
	}
}

// --- EncryptionRequired: the S4 downgrade-hole guard (defined now, enforced later) ---

func TestEncryptionRequired(t *testing.T) {
	cases := []struct {
		mode Mode
		want bool
	}{
		{ModePerDeviceKey, true},
		{ModeSharedKey, false},
		{ModeUnencrypted, false},
	}
	for _, c := range cases {
		got := Resolution{Mode: c.mode}.EncryptionRequired()
		if got != c.want {
			t.Errorf("mode %q: EncryptionRequired() = %v, want %v", c.mode, got, c.want)
		}
	}
}

// --- Key hygiene (doc 07 section 7.5: no raw key material in logs) ---

func TestKey_StringIsRedacted(t *testing.T) {
	k := Key(mustHex(fakePerDeviceKeyHex))
	s := k.String()
	if strings.Contains(strings.ToUpper(s), "0F1E2D") {
		t.Errorf("Key.String() leaked key material: %q", s)
	}
	if s == "" {
		t.Error("Key.String() must return a non-empty placeholder")
	}
}

func TestKey_ZeroWipesMaterial(t *testing.T) {
	k := Key(mustHex(fakePerDeviceKeyHex))
	k.Zero()
	for i, b := range k {
		if b != 0 {
			t.Fatalf("byte %d not zeroed: %#x", i, b)
		}
	}
}

func TestResolution_ZeroWipesKey(t *testing.T) {
	r := NewResolver(fakeDefaultKeyHex)
	got, err := r.Resolve(newDevice(fakePerDeviceKeyHex))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got.Zero()
	for i, b := range got.Key {
		if b != 0 {
			t.Fatalf("byte %d not zeroed: %#x", i, b)
		}
	}
}

// --- Mode/State validation helpers ---

func TestParseMode(t *testing.T) {
	cases := map[string]struct {
		want Mode
		ok   bool
	}{
		"PER_DEVICE_KEY": {ModePerDeviceKey, true},
		"SHARED_KEY":     {ModeSharedKey, true},
		"UNENCRYPTED":    {ModeUnencrypted, true},
		"per_device_key": {ModePerDeviceKey, true},
		"":               {"", false},
		"NONSENSE":       {"", false},
	}
	for in, c := range cases {
		got, ok := ParseMode(in)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("ParseMode(%q) = (%q, %v), want (%q, %v)", in, got, ok, c.want, c.ok)
		}
	}
}

func TestParseProvisionState(t *testing.T) {
	cases := map[string]struct {
		want ProvisionState
		ok   bool
	}{
		"ACTIVE":        {StateActive, true},
		"PENDING":       {StatePending, true},
		"UNPROVISIONED": {StateUnprovisioned, true},
		"RETIRED":       {StateRetired, true},
		"active":        {StateActive, true},
		"NONSENSE":      {"", false},
	}
	for in, c := range cases {
		got, ok := ParseProvisionState(in)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("ParseProvisionState(%q) = (%q, %v), want (%q, %v)", in, got, ok, c.want, c.ok)
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
