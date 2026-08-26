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
const testPendingKey = "A1B2C3D4E5F60718293A4B5C6D7E8F90"

// buildEncryptedRubixFrame produces a LoRaRAW frame carrying a minimal valid
// Rubix payload, encrypted under keyHex, with the RSSI/SNR trailer the radio
// bridge appends.
func buildEncryptedRubixFrame(t *testing.T, addr, keyHex string) string {
	t.Helper()
	key, err := hex.DecodeString(keyHex)
	if err != nil {
		t.Fatalf("decode key: %v", err)
	}
	// One positional uint16 datapoint — enough for the Rubix decoder.
	payload := []byte{0xA0, 0x00, 0x3C}
	frame, err := aesutils.Encrypt(addr, payload, key, utils.LORARAW_OPTS_UNCONFIRMED_UPLINK, 0x01)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	// Append the trailer the bridge adds: rssi, snr.
	frame = append(frame, 0x64, 0x0C)
	return strings.ToUpper(hex.EncodeToString(frame))
}

func newRubixDevice(t *testing.T, addr, manufacture string, tags ...*model.DeviceMetaTag) *model.Device {
	t.Helper()
	a := strings.ToUpper(addr)
	d := &model.Device{}
	d.Model = schema.DeviceModelRubixEncrypted // from the embedded CommonDevice
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

// A device still on its live key keeps working while a key is staged for it —
// the operator has not loaded the device yet.
func TestPendingKey_LiveKeyStillWorksDuringProvisioning(t *testing.T) {
	const addr = "5CC0D947"
	m := &Module{config: &Config{DefaultKey: testDefaultKey}}
	dev := newRubixDevice(t, addr, "", // no per-device key yet: on the shared key
		&model.DeviceMetaTag{Key: keymgmt.MetaKeyPendingKey, Value: testPendingKey},
		&model.DeviceMetaTag{Key: keymgmt.MetaKeyProvisionState, Value: string(keymgmt.StatePending)},
	)
	frame := buildEncryptedRubixFrame(t, addr, testDefaultKey)

	if res := dispatch(t, m, dev, frame); !res.OK {
		t.Fatal("frame under the LIVE key must still decode during the provisioning window")
	}
}

// Once the device has taken the staged key, its frames decrypt under the
// pending key — CE must accept them instead of losing contact.
func TestPendingKey_FrameUnderPendingKeyIsAccepted(t *testing.T) {
	const addr = "5CC0D947"
	m := &Module{config: &Config{DefaultKey: testDefaultKey}}
	dev := newRubixDevice(t, addr, "",
		&model.DeviceMetaTag{Key: keymgmt.MetaKeyPendingKey, Value: testPendingKey},
		&model.DeviceMetaTag{Key: keymgmt.MetaKeyProvisionState, Value: string(keymgmt.StatePending)},
	)
	frame := buildEncryptedRubixFrame(t, addr, testPendingKey)

	if res := dispatch(t, m, dev, frame); !res.OK {
		t.Fatal("frame under the PENDING key must be accepted during the provisioning window")
	}
}

// Without a staged key there is no second attempt: a frame under some other key
// is dropped. This is what stops the pending-key path becoming a general
// fallback.
func TestPendingKey_NoPendingKeyMeansNoSecondChance(t *testing.T) {
	const addr = "5CC0D947"
	m := &Module{config: &Config{DefaultKey: testDefaultKey}}
	dev := newRubixDevice(t, addr, "") // no pending key staged
	frame := buildEncryptedRubixFrame(t, addr, testPendingKey)

	if res := dispatch(t, m, dev, frame); res.OK {
		t.Fatal("a frame under an unknown key must be dropped when no key is staged")
	}
}

// A frame under a third, unrelated key is dropped even during the window.
func TestPendingKey_UnrelatedKeyIsStillDropped(t *testing.T) {
	const addr = "5CC0D947"
	const strangerKey = "FFEEDDCCBBAA99887766554433221100"
	m := &Module{config: &Config{DefaultKey: testDefaultKey}}
	dev := newRubixDevice(t, addr, "",
		&model.DeviceMetaTag{Key: keymgmt.MetaKeyPendingKey, Value: testPendingKey},
		&model.DeviceMetaTag{Key: keymgmt.MetaKeyProvisionState, Value: string(keymgmt.StatePending)},
	)
	frame := buildEncryptedRubixFrame(t, addr, strangerKey)

	if res := dispatch(t, m, dev, frame); res.OK {
		t.Fatal("a frame under an unrelated key must be dropped even during provisioning")
	}
}

// After confirmation the device is on its own key and the window is closed.
func TestPendingKey_AfterConfirmLiveKeyIsTheNewOne(t *testing.T) {
	const addr = "5CC0D947"
	m := &Module{config: &Config{DefaultKey: testDefaultKey}}
	// Post-confirm state: Manufacture holds the key, no pending tag.
	dev := newRubixDevice(t, addr, testPendingKey,
		&model.DeviceMetaTag{Key: keymgmt.MetaKeyProvisionState, Value: string(keymgmt.StateActive)},
		&model.DeviceMetaTag{Key: keymgmt.MetaKeyMode, Value: string(keymgmt.ModePerDeviceKey)},
	)
	frame := buildEncryptedRubixFrame(t, addr, testPendingKey)

	if res := dispatch(t, m, dev, frame); !res.OK {
		t.Fatal("after confirmation the promoted key must decode the device's frames")
	}

	// And the old shared key must no longer work for this device.
	oldFrame := buildEncryptedRubixFrame(t, addr, testDefaultKey)
	if res := dispatch(t, m, dev, oldFrame); res.OK {
		t.Fatal("after confirmation the shared key must NOT decode this device's frames")
	}
}
