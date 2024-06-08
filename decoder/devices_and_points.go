package decoder

import (
	"github.com/NubeIO/module-core-loraraw/schema"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"

	"strings"
)

type LoRaDeviceDescription struct {
	DeviceName      string
	Model           string
	SensorCode      string
	CheckLength     func(data string) bool
	Decode          func(data string, devDesc *LoRaDeviceDescription) (*CommonValues, interface{})
	GetPointsStruct func() interface{}
}

var NilLoRaDeviceDescription = LoRaDeviceDescription{
	DeviceName:      "",
	Model:           "",
	SensorCode:      "",
	CheckLength:     NilLoRaDeviceDescriptionCheckLength,
	Decode:          NilLoRaDeviceDescriptionDecode,
	GetPointsStruct: NilLoRaDeviceDescriptionGetPointsStruct,
}

func NilLoRaDeviceDescriptionCheckLength(data string) bool {
	return false
}

func NilLoRaDeviceDescriptionDecode(data string, devDesc *LoRaDeviceDescription) (*CommonValues, interface{}) {
	return &CommonValues{}, struct{}{}
}

func NilLoRaDeviceDescriptionGetPointsStruct() interface{} {
	return struct{}{}
}

var LoRaDeviceDescriptions = [...]LoRaDeviceDescription{
	{
		DeviceName:      "MicroEdge",
		Model:           schema.DeviceModelMicroEdgeV1,
		CheckLength:     CheckPayloadLengthME,
		Decode:          DecodeME,
		GetPointsStruct: GetPointsStructME,
	},
	{
		DeviceName:      "MicroEdge",
		Model:           schema.DeviceModelMicroEdgeV2,
		CheckLength:     CheckPayloadLengthME,
		Decode:          DecodeME,
		GetPointsStruct: GetPointsStructME,
	},
	{
		DeviceName:      "Droplet",
		Model:           schema.DeviceModelTHLM,
		CheckLength:     CheckPayloadLengthDroplet,
		Decode:          DecodeDropletTHLM,
		GetPointsStruct: GetPointsStructTHLM,
	},
	{
		DeviceName:      "Droplet",
		Model:           schema.DeviceModelTHL,
		CheckLength:     CheckPayloadLengthDroplet,
		Decode:          DecodeDropletTHL,
		GetPointsStruct: GetPointsStructTHL,
	},
	{
		DeviceName:      "Droplet",
		Model:           schema.DeviceModelTH,
		CheckLength:     CheckPayloadLengthDroplet,
		Decode:          DecodeDropletTH,
		GetPointsStruct: GetPointsStructTH,
	},
	{
		DeviceName:      "ZipHydroTap",
		Model:           schema.DeviceModelZiptHydroTap,
		CheckLength:     CheckPayloadLengthZHT,
		Decode:          DecodeZHT,
		GetPointsStruct: GetPointsStructZHT,
	},
	{
		DeviceName:      "Rubix",
		Model:           schema.DeviceModelRubix,
		CheckLength:     CheckPayloadLengthRubix,
		Decode:          DecodeRubix,
		GetPointsStruct: GetPointsStructRubix,
	},
}

func GetDeviceDescription(device *model.Device) *LoRaDeviceDescription {
	for _, dev := range LoRaDeviceDescriptions {
		if strings.EqualFold(device.Model, dev.Model) {
			return &dev
		}
	}
	return &NilLoRaDeviceDescription
}

func GetDevicePointsStruct(device *model.Device) interface{} {
	return GetDeviceDescription(device).GetPointsStruct()
}
