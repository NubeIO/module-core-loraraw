package pkg

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/NubeIO/module-core-loraraw/codec"
	"github.com/NubeIO/module-core-loraraw/schema"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
)

// TestEncryptedRubixLengthFalsePositive is a regression test for the LoRaRAW
// encrypted/plaintext classification bug.
//
// This exact frame (device "Optical-Test", address 65C0640D) is AES-encrypted,
// but a ciphertext byte at the inner-length offset happens to equal the value
// that the old length-only heuristic (isUnencryptedLoRaRAW) expected for a
// plaintext frame. The dispatcher therefore skipped decryption and ran the raw
// ciphertext through the bit-stream decoder, publishing garbage points
// (unknown-*, uint_64-8 = 1.7e19, etc.).
//
// With the CMAC-first classification, the frame decrypts (CMAC verifies) and
// the real positional UVP points are emitted. If this test ever regresses to
// the garbage decode, the asserted UVP points will be missing.
func TestEncryptedRubixLengthFalsePositive(t *testing.T) {
	test = t

	// The encrypted/plaintext classification lives in dispatchFrame's
	// IsLoRaRAW branch, ahead of model-specific codec dispatch, so the fix is
	// model-agnostic. Rubix, UART and RubixEncrypted are IsLoRaRAW devices sharing
	// the same DecodeRubixUplink decoder; assert all three to lock that in.
	models := []string{schema.DeviceModelRubix, schema.DeviceModelUART, schema.DeviceModelRubixEncrypted}
	for _, deviceModel := range models {
		deviceModel := deviceModel
		t.Run(deviceModel, func(t *testing.T) {
			test = t
			addr := "65C0640D"
			mockDevice := &model.Device{
				Name: "Optical-Test",
				CommonDevice: model.CommonDevice{
					Model:       deviceModel,
					AddressUUID: &addr,
				},
			}

			tests := []TestStruct{
				{
					Name: deviceModel + "-Encrypted-LengthFalsePositive",
					Data: "65C0640DA98521CC47B800BF4F2E90E4014F5279F207180C56A29EE9604CE987A1BA825351BDEF154126",
					Values: []TestPoint{
						{"UVP-1", 600},
						{"UVP-2", 3.8},
						{"UVP-3", 6423},
						{"UVP-6", 5},
						{"rssi", -65},
						{"snr", 9.5},
					},
					MetaTags: []*model.DeviceMetaTag{},
				},
			}

			runDispatchTests(tests, mockDevice, t)
		})
	}
}

// TestCorruptedEncryptedRubixDropped locks in the fix for the corrupted-
// ciphertext → false-plaintext → garbage-points leak.
//
// It reuses the AES-encrypted Rubix fixture from the test above, whose
// ciphertext byte at the inner-length offset makes it satisfy the plaintext
// length heuristic (isUnencryptedLoRaRAW). We then flip a *ciphertext* byte
// (not the length byte at index 6, not the trailing rssi/snr) so the frame
// still passes isUnencryptedLoRaRAW but now fails CMAC — exactly what a weak-RF
// bit-error does to a genuine encrypted frame.
//
// This is the ambiguous overlap the fix resolves: the frame is BOTH
// encryption-shaped (block-aligned inner) AND plaintext-length-valid. Because a
// block-aligned frame that fails CMAC is a corrupted ciphertext (not
// plaintext), dispatchFrame must DROP it (case 3) rather than decode the
// ciphertext into garbage points. Before the fix this produced a burst of junk
// points (unknown-*, huge values).
//
// Asserted for both Rubix (plaintext-allowed: the drop comes from the
// not-encryption-shaped guard) and RubixEncrypted (encryption-only: the drop comes
// from the model policy) — both must refuse to decode this frame.
func TestCorruptedEncryptedRubixDropped(t *testing.T) {
	test = t

	const encryptedFixture = "65C0640DA98521CC47B800BF4F2E90E4014F5279F207180C56A29EE9604CE987A1BA825351BDEF154126"
	b, err := hex.DecodeString(encryptedFixture)
	if err != nil {
		t.Fatalf("decode fixture: %s", err)
	}
	b[12] ^= 0xFF // corrupt a ciphertext byte → CMAC fails; length byte (idx 6) untouched

	// Precondition: the corrupted frame must sit in the overlap zone — it must
	// look like plaintext to the length heuristic AND be encryption-shaped —
	// otherwise it would be dropped for the wrong reason and the test wouldn't
	// exercise the leak.
	if !isUnencryptedLoRaRAW(b) {
		t.Fatalf("corrupted fixture no longer satisfies isUnencryptedLoRaRAW; cannot exercise the leak")
	}
	if !isEncryptionShaped(b) {
		t.Fatalf("corrupted fixture is not encryption-shaped; cannot exercise the leak")
	}
	corrupted := strings.ToUpper(hex.EncodeToString(b))

	for _, deviceModel := range []string{schema.DeviceModelRubix, schema.DeviceModelRubixEncrypted} {
		deviceModel := deviceModel
		t.Run(deviceModel, func(t *testing.T) {
			test = t
			addr := "65C0640D"
			mockDevice := &model.Device{
				Name: "Optical-Test",
				CommonDevice: model.CommonDevice{
					Model:       deviceModel,
					AddressUUID: &addr,
				},
			}
			m := &Module{config: &Config{DefaultKey: testDefaultKey}}

			got := map[string]float64{}
			capture := func(name string, value float64, _ *model.Device, _ *codec.LoRaDeviceDescription) error {
				got[name] = value
				return nil
			}

			res := m.dispatchFrame(
				corrupted,
				newMockGetDevice(mockDevice, addr),
				capture, noopPointErr, noopMetaTags,
				noopWrittenOK, noopWrittenErr,
			)

			if res.OK {
				t.Errorf("corrupted encrypted %s frame was accepted (OK=true); expected drop", deviceModel)
			}
			if len(got) != 0 {
				t.Errorf("corrupted encrypted %s frame produced %d point(s), expected 0 (garbage leak): %v", deviceModel, len(got), got)
			}
		})
	}
}
