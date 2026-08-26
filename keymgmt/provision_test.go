package keymgmt

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
)

// --- GenerateKey ---

func TestGenerateKey_LengthAndRandomness(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 32; i++ {
		k, err := GenerateKey()
		if err != nil {
			t.Fatalf("GenerateKey: %v", err)
		}
		if len(k) != KeyLen {
			t.Fatalf("key length = %d, want %d", len(k), KeyLen)
		}
		h := hex.EncodeToString(k)
		if seen[h] {
			t.Fatalf("GenerateKey returned a duplicate key on iteration %d", i)
		}
		seen[h] = true
	}
}

func TestGenerateKey_NotAllZero(t *testing.T) {
	k, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	if bytes.Equal(k, make([]byte, KeyLen)) {
		t.Fatal("GenerateKey returned an all-zero key")
	}
}

func TestGenerateKeyHex_IsUppercaseAndParses(t *testing.T) {
	h, err := GenerateKeyHex()
	if err != nil {
		t.Fatalf("GenerateKeyHex: %v", err)
	}
	if len(h) != KeyLen*2 {
		t.Fatalf("hex length = %d, want %d", len(h), KeyLen*2)
	}
	for _, r := range h {
		if !((r >= '0' && r <= '9') || (r >= 'A' && r <= 'F')) {
			t.Fatalf("hex contains non-uppercase-hex rune %q in %q", r, h)
		}
	}
	if _, err := hex.DecodeString(h); err != nil {
		t.Fatalf("hex does not decode: %v", err)
	}
}

// --- Pending key: the rotation window (doc 08 section 8.2) ---

func TestResolve_PendingKeyIsExposedDuringProvisioning(t *testing.T) {
	r := NewResolver(fakeDefaultKeyHex)
	d := newDevice(fakePerDeviceKeyHex,
		metaTag(MetaKeyPendingKey, fakePendingKeyHex),
		metaTag(MetaKeyProvisionState, string(StatePending)),
	)
	got, err := r.Resolve(d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Current key stays active so a device that has NOT yet taken the new
	// key keeps working.
	want, _ := hex.DecodeString(fakePerDeviceKeyHex)
	if !bytes.Equal(got.Key, want) {
		t.Errorf("Key = %X, want current key %X", got.Key, want)
	}
	wantPending, _ := hex.DecodeString(fakePendingKeyHex)
	if !bytes.Equal(got.PendingKey, wantPending) {
		t.Errorf("PendingKey = %X, want %X", got.PendingKey, wantPending)
	}
	if got.State != StatePending {
		t.Errorf("State = %q, want %q", got.State, StatePending)
	}
}

func TestResolve_NoPendingKeyWhenNotProvisioning(t *testing.T) {
	r := NewResolver(fakeDefaultKeyHex)
	got, err := r.Resolve(newDevice(fakePerDeviceKeyHex))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.PendingKey != nil {
		t.Errorf("PendingKey = %X, want nil", got.PendingKey)
	}
}

func TestResolve_InvalidPendingKeyIsIgnoredNotFatal(t *testing.T) {
	r := NewResolver(fakeDefaultKeyHex)
	d := newDevice(fakePerDeviceKeyHex, metaTag(MetaKeyPendingKey, "NOTHEX"))
	got, err := r.Resolve(d)
	// A corrupt pending key must not break the device's normal traffic.
	if err != nil {
		t.Fatalf("a bad pending key must not fail Resolve, got: %v", err)
	}
	if got.PendingKey != nil {
		t.Errorf("PendingKey = %X, want nil for corrupt value", got.PendingKey)
	}
	want, _ := hex.DecodeString(fakePerDeviceKeyHex)
	if !bytes.Equal(got.Key, want) {
		t.Errorf("current key must be unaffected")
	}
}

func TestResolution_ZeroWipesPendingKeyToo(t *testing.T) {
	r := NewResolver(fakeDefaultKeyHex)
	d := newDevice(fakePerDeviceKeyHex, metaTag(MetaKeyPendingKey, fakePendingKeyHex))
	got, _ := r.Resolve(d)
	got.Zero()
	for i, b := range got.PendingKey {
		if b != 0 {
			t.Fatalf("pending key byte %d not zeroed: %#x", i, b)
		}
	}
}

// --- Provision plans: pure functions producing the meta tags to persist ---

func TestBeginProvision_KeepsCurrentKeyAndStagesNewOne(t *testing.T) {
	d := newDevice(fakePerDeviceKeyHex)
	plan, err := BeginProvision(d, fakePendingKeyHex)
	if err != nil {
		t.Fatalf("BeginProvision: %v", err)
	}
	// The device's live key must NOT change yet — it has not taken the new
	// key, so overwriting Manufacture now would lose contact.
	if plan.Manufacture != "" {
		t.Errorf("Manufacture = %q, want empty (unchanged) during BeginProvision", plan.Manufacture)
	}
	assertTag(t, plan.MetaTags, MetaKeyPendingKey, fakePendingKeyHex)
	assertTag(t, plan.MetaTags, MetaKeyProvisionState, string(StatePending))
}

func TestBeginProvision_RejectsBadKey(t *testing.T) {
	d := newDevice("")
	for _, bad := range []string{"", "NOTHEX", "0011223344556677"} {
		if _, err := BeginProvision(d, bad); err == nil {
			t.Errorf("BeginProvision(%q) should fail", bad)
		}
	}
}

func TestBeginProvision_RejectsRetiredDevice(t *testing.T) {
	d := newDevice(fakePerDeviceKeyHex, metaTag(MetaKeyProvisionState, string(StateRetired)))
	if _, err := BeginProvision(d, fakePendingKeyHex); err == nil {
		t.Error("provisioning a RETIRED device should fail")
	}
}

func TestConfirmProvision_PromotesPendingKey(t *testing.T) {
	d := newDevice(fakePerDeviceKeyHex,
		metaTag(MetaKeyPendingKey, fakePendingKeyHex),
		metaTag(MetaKeyProvisionState, string(StatePending)),
	)
	plan, err := ConfirmProvision(d)
	if err != nil {
		t.Fatalf("ConfirmProvision: %v", err)
	}
	if plan.Manufacture != fakePendingKeyHex {
		t.Errorf("Manufacture = %q, want promoted pending key %q", plan.Manufacture, fakePendingKeyHex)
	}
	assertTag(t, plan.MetaTags, MetaKeyProvisionState, string(StateActive))
	assertTag(t, plan.MetaTags, MetaKeyMode, string(ModePerDeviceKey))
	// Pending key must be cleared so it cannot be reused.
	assertTag(t, plan.MetaTags, MetaKeyPendingKey, "")
}

func TestConfirmProvision_FailsWithoutPendingKey(t *testing.T) {
	d := newDevice(fakePerDeviceKeyHex)
	if _, err := ConfirmProvision(d); err == nil {
		t.Error("ConfirmProvision without a pending key should fail")
	}
}

func TestAbortProvision_ClearsPendingKeyAndKeepsCurrent(t *testing.T) {
	d := newDevice(fakePerDeviceKeyHex,
		metaTag(MetaKeyPendingKey, fakePendingKeyHex),
		metaTag(MetaKeyProvisionState, string(StatePending)),
	)
	plan, err := AbortProvision(d)
	if err != nil {
		t.Fatalf("AbortProvision: %v", err)
	}
	if plan.Manufacture != "" {
		t.Errorf("Manufacture = %q, want unchanged", plan.Manufacture)
	}
	assertTag(t, plan.MetaTags, MetaKeyPendingKey, "")
	// Device had a per-device key already, so it returns to ACTIVE.
	assertTag(t, plan.MetaTags, MetaKeyProvisionState, string(StateActive))
}

func TestAbortProvision_UnprovisionedDeviceReturnsToUnprovisioned(t *testing.T) {
	d := newDevice("", // never had a per-device key
		metaTag(MetaKeyPendingKey, fakePendingKeyHex),
		metaTag(MetaKeyProvisionState, string(StatePending)),
	)
	plan, err := AbortProvision(d)
	if err != nil {
		t.Fatalf("AbortProvision: %v", err)
	}
	assertTag(t, plan.MetaTags, MetaKeyProvisionState, string(StateUnprovisioned))
}

func TestRetireDevice_ClearsKeysAndTombstones(t *testing.T) {
	d := newDevice(fakePerDeviceKeyHex)
	plan, err := RetireDevice(d)
	if err != nil {
		t.Fatalf("RetireDevice: %v", err)
	}
	assertTag(t, plan.MetaTags, MetaKeyProvisionState, string(StateRetired))
	assertTag(t, plan.MetaTags, MetaKeyPendingKey, "")
}

func TestPlans_RejectNilDevice(t *testing.T) {
	if _, err := BeginProvision(nil, fakePendingKeyHex); err == nil {
		t.Error("BeginProvision(nil) should fail")
	}
	if _, err := ConfirmProvision(nil); err == nil {
		t.Error("ConfirmProvision(nil) should fail")
	}
	if _, err := AbortProvision(nil); err == nil {
		t.Error("AbortProvision(nil) should fail")
	}
	if _, err := RetireDevice(nil); err == nil {
		t.Error("RetireDevice(nil) should fail")
	}
}

// --- helpers ---

const fakePendingKeyHex = "A1B2C3D4E5F60718293A4B5C6D7E8F90"

func assertTag(t *testing.T, tags []*model.DeviceMetaTag, key, want string) {
	t.Helper()
	for _, tag := range tags {
		if tag.Key == key {
			if tag.Value != want {
				t.Errorf("meta tag %q = %q, want %q", key, tag.Value, want)
			}
			return
		}
	}
	t.Errorf("meta tag %q not present in plan", key)
}
