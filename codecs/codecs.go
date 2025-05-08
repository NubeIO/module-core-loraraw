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
	},
	{
		DeviceName:     "Rubix",
		Model:          schema.DeviceModelRubix,
		CheckLength:    rubixDataEncoding.CheckPayloadLengthRubix,
		DecodeUplink:   rubixDataEncoding.DecodeRubixUplink,
		DecodeResponse: rubixDataEncoding.DecodeRubixResponse,
		GetPointNames:  rubixDataEncoding.GetRubixPointNames,
		IsLoRaRAW:      true,
	},
}
