package codecs

import (
	"github.com/NubeIO/module-core-loraraw/codec"
	"github.com/NubeIO/module-core-loraraw/codecs/legacyDecoders"
	"github.com/NubeIO/module-core-loraraw/codecs/rubixDataEncoding"
	"github.com/NubeIO/module-core-loraraw/schema"
)

var LoRaDeviceDescriptions = []codec.LoRaDeviceDescription{
	{
		// LEGACY DEVICE. PLS REMOVE IN FUTURE
		DeviceName:    "MicroEdge",
		Model:         schema.DeviceModelMicroEdgeV1,
		CheckLength:   legacyDecoders.CheckPayloadLengthME,
		DecodeUplink:  legacyDecoders.DecodeME,
		GetPointNames: legacyDecoders.GetMePointNames,
		IsLoRaRAW:     false,
	},
	{
		// LEGACY DEVICE. PLS REMOVE IN FUTURE
		DeviceName:    "MicroEdge",
		Model:         schema.DeviceModelMicroEdgeV2,
		CheckLength:   legacyDecoders.CheckPayloadLengthME,
		DecodeUplink:  legacyDecoders.DecodeME,
		GetPointNames: legacyDecoders.GetMePointNames,
		IsLoRaRAW:     false,
	},
	{
		// LEGACY DEVICE. PLS REMOVE IN FUTURE
		DeviceName:    "Droplet",
		Model:         schema.DeviceModelTHLM,
		CheckLength:   legacyDecoders.CheckPayloadLengthDroplet,
		DecodeUplink:  legacyDecoders.DecodeDropletTHLM,
		GetPointNames: legacyDecoders.GetTHLMPointNames,
		IsLoRaRAW:     false,
	},
	{
		// LEGACY DEVICE. PLS REMOVE IN FUTURE
		DeviceName:    "Droplet",
		Model:         schema.DeviceModelTHL,
		CheckLength:   legacyDecoders.CheckPayloadLengthDroplet,
		DecodeUplink:  legacyDecoders.DecodeDropletTHL,
		GetPointNames: legacyDecoders.GetTHLPointNames,
		IsLoRaRAW:     false,
	},
	{
		// LEGACY DEVICE. PLS REMOVE IN FUTURE
		DeviceName:    "Droplet",
		Model:         schema.DeviceModelTH,
		CheckLength:   legacyDecoders.CheckPayloadLengthDroplet,
		DecodeUplink:  legacyDecoders.DecodeDropletTH,
		GetPointNames: legacyDecoders.GetTHPointNames,
		IsLoRaRAW:     false,
	},
	{
		// LEGACY DEVICE. PLS REMOVE IN FUTURE
		DeviceName:    "ZipHydroTap",
		Model:         schema.DeviceModelZiptHydroTap,
		CheckLength:   legacyDecoders.CheckPayloadLengthZHT,
		DecodeUplink:  legacyDecoders.DecodeZHT,
		GetPointNames: legacyDecoders.GetZHTPointNames,
		IsLoRaRAW:     true,
		// ZHT has genuine unencrypted deployments; allow the plaintext path.
		// Its strict CheckPayloadLengthZHT (packet-version × payload-type ×
		// exact length) plus the not-encryption-shaped guard in dispatchFrame
		// keep corrupt frames out. UART and RubixEncrypted leave this false:
		// they are encryption-only, so a CMAC failure is always dropped.
		AllowUnencrypted: true,
	},
	{
		DeviceName:           "Rubix",
		Model:                schema.DeviceModelRubix,
		CheckLength:          rubixDataEncoding.CheckPayloadLengthRubix,
		DecodeUplink:         rubixDataEncoding.DecodeRubixUplink,
		DecodeResponse:       rubixDataEncoding.DecodeRubixResponse,
		EncodeRequestMessage: rubixDataEncoding.EncodeRequestMessage,
		GetPointNames:        rubixDataEncoding.GetRubixPointNames,
		IsLoRaRAW:            true,
		// Rubix covers mixed field populations, including genuinely plaintext
		// devices (e.g. Dorma door nodes). Keep the plaintext path open; the
		// not-encryption-shaped guard in dispatchFrame still rejects corrupted
		// ciphertext. Devices known to encrypt should use the
		// RubixEncrypted model instead, which is encryption-only.
		AllowUnencrypted: true,
	},
	{
		// Encryption-only: UART firmware always encrypts (AES-CBC + CMAC), so
		// any frame that fails CMAC is corruption and is dropped — never
		// re-interpreted as plaintext.
		DeviceName:           "UART",
		Model:                schema.DeviceModelUART,
		CheckLength:          rubixDataEncoding.CheckPayloadLengthRubix,
		DecodeUplink:         rubixDataEncoding.DecodeRubixUplink,
		DecodeResponse:       rubixDataEncoding.DecodeRubixResponse,
		EncodeRequestMessage: rubixDataEncoding.EncodeRequestMessage,
		GetPointNames:        rubixDataEncoding.GetRubixPointNames,
		IsLoRaRAW:            true,
	},
	{
		// Encryption-only Rubix (e.g. optical power meter). Same wire format
		// and codec as Rubix, but for devices whose firmware always encrypts
		// (AES-CBC + CMAC): a CMAC failure is corruption and is dropped. This
		// is what stops weak-RF corrupted frames from being decoded as
		// plaintext into garbage points. Encrypting devices previously
		// provisioned as model "Rubix" should be re-provisioned as
		// "RubixEncrypted" to get this guarantee.
		DeviceName:           "RubixEncrypted",
		Model:                schema.DeviceModelRubixEncrypted,
		CheckLength:          rubixDataEncoding.CheckPayloadLengthRubix,
		DecodeUplink:         rubixDataEncoding.DecodeRubixUplink,
		DecodeResponse:       rubixDataEncoding.DecodeRubixResponse,
		EncodeRequestMessage: rubixDataEncoding.EncodeRequestMessage,
		GetPointNames:        rubixDataEncoding.GetRubixPointNames,
		IsLoRaRAW:            true,
	},
}
