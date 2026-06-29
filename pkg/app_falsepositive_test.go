package pkg

import (
	"testing"

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
	// model-agnostic. Both Rubix and UART are IsLoRaRAW devices sharing the
	// same DecodeRubixUplink decoder; assert both to lock that in.
	models := []string{schema.DeviceModelRubix, schema.DeviceModelUART}
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
