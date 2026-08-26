package pkg

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/NubeIO/module-core-loraraw/aesutils"
	"github.com/NubeIO/module-core-loraraw/codec"
	"github.com/NubeIO/module-core-loraraw/keymgmt"
	"github.com/NubeIO/module-core-loraraw/schema"
	"github.com/NubeIO/module-core-loraraw/utils"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
)

// Clearly-fake test vector, never a real key.
const testOwnKey = "A1B2C3D4E5F60718293A4B5C6D7E8F90"

// buildPlaintextFrame builds a plaintext LoRaRAW frame:
// [addr:4][opts:1][nonce:1][len:1][payload:len][rssi:1][snr:1]
func buildPlaintextFrame(t *testing.T, addr string, payload []byte) string {
	t.Helper()
	a, err := hex.DecodeString(addr)
	if err != nil {
		t.Fatalf("decode addr: %v", err)
	}
	f := append([]byte{}, a...)
	f = append(f, 0x00, 0x01, byte(len(payload)))
	f = append(f, payload...)
	f = append(f, 0x64, 0x0C) // rssi, snr trailer
	return strings.ToUpper(hex.EncodeToString(f))
}

func buildEncryptedFrame(t *testing.T, addr, keyHex string) string {
	t.Helper()
	key, err := hex.DecodeString(keyHex)
	if err != nil {
		t.Fatalf("decode key: %v", err)
	}
	payload := []byte{0xA0, 0x00, 0x3C} // one positional uint16 datapoint
	f, err := aesutils.Encrypt(addr, payload, key, utils.LORARAW_OPTS_UNCONFIRMED_UPLINK, 0x01)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	f = append(f, 0x64, 0x0C)
	return strings.ToUpper(hex.EncodeToString(f))
}

// newDeviceAllowingPlaintext returns a device on the Rubix model, which sets
// AllowUnencrypted=true - the model the downgrade hole applied to.
func newDeviceAllowingPlaintext(t *testing.T, addr, manufacture string, tags ...*model.DeviceMetaTag) *model.Device {
	t.Helper()
	a := strings.ToUpper(addr)
	d := &model.Device{}
	d.Model = schema.DeviceModelRubix // from the embedded CommonDevice
	d.AddressUUID = &a
	d.Manufacture = manufacture
	d.MetaTags = tags
	return d
}

func dispatch(t *testing.T, m *Module, dev *model.Device, frameHex string) DispatchResult {
	t.Helper()
	wireAddr, err := codec.DecodeAddressHex(frameHex)
	if err != nil {
		t.Fatalf("decode wire address: %v", err)
	}
	return m.dispatchFrame(
		frameHex,
		newMockGetDevice(dev, strings.ToUpper(wireAddr)),
		func(string, float64, *model.Device, *codec.LoRaDeviceDescription) error { return nil },
		noopPointErr, noopMetaTags, noopWrittenOK, noopWrittenErr,
	)
}

// THE DOWNGRADE HOLE (G3): a device with its own key must never accept a
// plaintext frame, even though its model allows plaintext in general. The
// device address is cleartext on air, so anyone who hears one frame could
// otherwise forge plaintext and bypass encryption entirely.
func TestDowngrade_DeviceWithOwnKeyRejectsPlaintext(t *testing.T) {
	const addr = "5CC0D947"
	m := &Module{config: &Config{DefaultKey: testDefaultKey}}
	dev := newDeviceAllowingPlaintext(t, addr, testOwnKey)
	frame := buildPlaintextFrame(t, addr, []byte{0xA0, 0x00, 0x3C})

	if res := dispatch(t, m, dev, frame); res.OK {
		t.Fatal("DOWNGRADE HOLE: a device with its own key must NOT accept a plaintext frame")
	}
}

// Devices still on the shared key keep their old behaviour - this change must
// not break the existing fleet.
func TestDowngrade_SharedKeyDeviceStillAcceptsPlaintext(t *testing.T) {
	const addr = "3CC094FD"
	m := &Module{config: &Config{DefaultKey: testDefaultKey}}
	dev := newDeviceAllowingPlaintext(t, addr, "") // no own key
	frame := buildPlaintextFrame(t, addr, []byte{0xA0, 0x00, 0x3C})

	if res := dispatch(t, m, dev, frame); !res.OK {
		t.Fatal("a shared-key device must keep accepting plaintext (no behaviour change)")
	}
}

// Pasting the SHARED key into the Device Key box is not provisioning. It must
// not flip the device into encryption-required, because that would silently
// break a device the operator believes they just secured.
func TestDowngrade_SharedKeyPastedIntoDeviceKeyBoxIsStillShared(t *testing.T) {
	const addr = "3CC094FD"
	m := &Module{config: &Config{DefaultKey: testDefaultKey}}
	dev := newDeviceAllowingPlaintext(t, addr, testDefaultKey) // shared key pasted in
	frame := buildPlaintextFrame(t, addr, []byte{0xA0, 0x00, 0x3C})

	if res := dispatch(t, m, dev, frame); !res.OK {
		t.Fatal("pasting the shared key must not turn the device into encryption-required")
	}
}

// An explicit UNENCRYPTED opt-in keeps working.
func TestDowngrade_ExplicitUnencryptedDeviceAcceptsPlaintext(t *testing.T) {
	const addr = "3CC094FD"
	m := &Module{config: &Config{DefaultKey: testDefaultKey}}
	dev := newDeviceAllowingPlaintext(t, addr, "",
		&model.DeviceMetaTag{Key: keymgmt.MetaKeyMode, Value: string(keymgmt.ModeUnencrypted)})
	frame := buildPlaintextFrame(t, addr, []byte{0xA0, 0x00, 0x3C})

	if res := dispatch(t, m, dev, frame); !res.OK {
		t.Fatal("a device explicitly set to UNENCRYPTED must accept plaintext")
	}
}

// Closing the hole must not stop genuine encrypted traffic.
func TestDowngrade_DeviceWithOwnKeyStillDecryptsItsFrames(t *testing.T) {
	const addr = "5CC0D947"
	m := &Module{config: &Config{DefaultKey: testDefaultKey}}
	dev := newDeviceAllowingPlaintext(t, addr, testOwnKey)
	frame := buildEncryptedFrame(t, addr, testOwnKey)

	if res := dispatch(t, m, dev, frame); !res.OK {
		t.Fatal("a device with its own key must still decode its own encrypted frames")
	}
}

// After a device gets its own key, the shared key must no longer decode it.
func TestDowngrade_SharedKeyNoLongerDecodesProvisionedDevice(t *testing.T) {
	const addr = "5CC0D947"
	m := &Module{config: &Config{DefaultKey: testDefaultKey}}
	dev := newDeviceAllowingPlaintext(t, addr, testOwnKey)
	frame := buildEncryptedFrame(t, addr, testDefaultKey) // encrypted with the OLD shared key

	if res := dispatch(t, m, dev, frame); res.OK {
		t.Fatal("once a device has its own key, the shared key must NOT decode its frames")
	}
}

// A dash-separated key (what the ESP32 console prints) pasted into the UI must
// work - otherwise an operator copying from `param_get 0x0220` is stuck.
func TestDowngrade_DashSeparatedKeyFromEsp32ConsoleWorks(t *testing.T) {
	const addr = "C3C0A660"
	dashed := strings.Join([]string{
		"A1", "B2", "C3", "D4", "E5", "F6", "07", "18",
		"29", "3A", "4B", "5C", "6D", "7E", "8F", "90",
	}, "-")
	m := &Module{config: &Config{DefaultKey: testDefaultKey}}
	dev := newDeviceAllowingPlaintext(t, addr, dashed)
	frame := buildEncryptedFrame(t, addr, testOwnKey) // same key, plain hex

	if res := dispatch(t, m, dev, frame); !res.OK {
		t.Fatal("a dash-separated key from the ESP32 console must be accepted")
	}
}
