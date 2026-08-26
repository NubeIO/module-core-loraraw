package pkg

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/NubeIO/module-core-loraraw/keymgmt"
	"github.com/NubeIO/module-core-loraraw/schema"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
)

// buildPlaintextZHTFrame builds a plaintext LoRaRAW frame for the ZipHydroTap
// model, which is one of the models that legitimately allows plaintext
// (AllowUnencrypted=true) — exactly the case the downgrade hole exploited.
//
// Layout: [addr:4][opts:1][nonce:1][len:1][payload:len][rssi:1][snr:1]
func buildPlaintextFrame(t *testing.T, addr string, payload []byte) string {
	t.Helper()
	a, err := hex.DecodeString(addr)
	if err != nil {
		t.Fatalf("decode addr: %v", err)
	}
	frame := append([]byte{}, a...)
	frame = append(frame, 0x00, 0x01, byte(len(payload))) // opts, nonce, len
	frame = append(frame, payload...)
	frame = append(frame, 0x64, 0x0C) // rssi, snr trailer
	return strings.ToUpper(hex.EncodeToString(frame))
}

func newZHTDevice(t *testing.T, addr, manufacture string, tags ...*model.DeviceMetaTag) *model.Device {
	t.Helper()
	a := strings.ToUpper(addr)
	d := &model.Device{}
	d.Model = schema.DeviceModelRubix // AllowUnencrypted = true
	d.AddressUUID = &a
	d.Manufacture = manufacture
	d.MetaTags = tags
	return d
}

// THE DOWNGRADE HOLE (G3): before S4, a device with its own key still accepted
// a well-formed PLAINTEXT frame, because its model allows plaintext. Anyone who
// knew the (cleartext) address could bypass encryption entirely.
func TestDowngrade_PerDeviceKeyDeviceRejectsPlaintext(t *testing.T) {
	const addr = "5CC0D947"
	m := &Module{config: &Config{DefaultKey: testDefaultKey}}
	dev := newZHTDevice(t, addr, testPendingKey, // has its own key => PER_DEVICE_KEY
		&model.DeviceMetaTag{Key: keymgmt.MetaKeyMode, Value: string(keymgmt.ModePerDeviceKey)},
		&model.DeviceMetaTag{Key: keymgmt.MetaKeyProvisionState, Value: string(keymgmt.StateActive)},
	)
	frame := buildPlaintextFrame(t, addr, []byte{0xA0, 0x00, 0x3C})

	if res := dispatch(t, m, dev, frame); res.OK {
		t.Fatal("DOWNGRADE HOLE: a device with its own key must NOT accept a plaintext frame")
	}
}

// A device still on the shared key keeps its old behaviour: plaintext is
// accepted for models that allow it. S4 must not break the existing fleet.
func TestDowngrade_SharedKeyDeviceStillAcceptsPlaintext(t *testing.T) {
	const addr = "3CC094FD"
	m := &Module{config: &Config{DefaultKey: testDefaultKey}}
	dev := newZHTDevice(t, addr, "") // no per-device key => SHARED_KEY
	frame := buildPlaintextFrame(t, addr, []byte{0xA0, 0x00, 0x3C})

	if res := dispatch(t, m, dev, frame); !res.OK {
		t.Fatal("a shared-key device must keep accepting plaintext (no behaviour change)")
	}
}

// A device explicitly marked UNENCRYPTED is an opt-in, so plaintext is fine.
func TestDowngrade_ExplicitlyUnencryptedDeviceAcceptsPlaintext(t *testing.T) {
	const addr = "3CC094FD"
	m := &Module{config: &Config{DefaultKey: testDefaultKey}}
	dev := newZHTDevice(t, addr, "",
		&model.DeviceMetaTag{Key: keymgmt.MetaKeyMode, Value: string(keymgmt.ModeUnencrypted)},
	)
	frame := buildPlaintextFrame(t, addr, []byte{0xA0, 0x00, 0x3C})

	if res := dispatch(t, m, dev, frame); !res.OK {
		t.Fatal("a device explicitly set to UNENCRYPTED must accept plaintext")
	}
}

// Closing the hole must not stop a per-device-key device's genuine ENCRYPTED
// traffic from working.
func TestDowngrade_PerDeviceKeyDeviceStillAcceptsItsEncryptedFrames(t *testing.T) {
	const addr = "5CC0D947"
	m := &Module{config: &Config{DefaultKey: testDefaultKey}}
	dev := newZHTDevice(t, addr, testPendingKey,
		&model.DeviceMetaTag{Key: keymgmt.MetaKeyMode, Value: string(keymgmt.ModePerDeviceKey)},
	)
	frame := buildEncryptedRubixFrame(t, addr, testPendingKey)

	if res := dispatch(t, m, dev, frame); !res.OK {
		t.Fatal("a per-device-key device must still decode its own encrypted frames")
	}
}

// A device mid-provisioning (has a pending key but no own key yet) is still on
// the shared key, so plaintext must not be blocked prematurely.
func TestDowngrade_PendingProvisioningDoesNotBlockPlaintextYet(t *testing.T) {
	const addr = "3CC094FD"
	m := &Module{config: &Config{DefaultKey: testDefaultKey}}
	dev := newZHTDevice(t, addr, "", // no own key yet
		&model.DeviceMetaTag{Key: keymgmt.MetaKeyPendingKey, Value: testPendingKey},
		&model.DeviceMetaTag{Key: keymgmt.MetaKeyProvisionState, Value: string(keymgmt.StatePending)},
	)
	frame := buildPlaintextFrame(t, addr, []byte{0xA0, 0x00, 0x3C})

	if res := dispatch(t, m, dev, frame); !res.OK {
		t.Fatal("a device mid-provisioning is still on the shared key; plaintext must not be blocked yet")
	}
}
