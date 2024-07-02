package decoder

import (
	"errors"

	"github.com/NubeIO/module-core-loraraw/schema"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"

	"strings"
)

type LoRaDeviceDescription struct {
	DeviceName    string
	Model         string
	SensorCode    string
	CheckLength   func(data string) bool
	Decode        func(data string, devDesc *LoRaDeviceDescription, device *model.Device) error
	GetPointNames func() []string
}

var NilLoRaDeviceDescription = LoRaDeviceDescription{
	DeviceName:    "",
	Model:         "",
	SensorCode:    "",
	CheckLength:   NilLoRaDeviceDescriptionCheckLength,
	Decode:        NilLoRaDeviceDescriptionDecode,
	GetPointNames: NilLoRaDeviceDescriptionGetPointsStruct,
}

func NilLoRaDeviceDescriptionCheckLength(data string) bool {
	return false
}

func NilLoRaDeviceDescriptionDecode(data string, devDesc *LoRaDeviceDescription, device *model.Device) error {
	return errors.New("nil decode function called")
}

func NilLoRaDeviceDescriptionGetPointsStruct() []string {
	return []string{}
}

var LoRaDeviceDescriptions = [...]LoRaDeviceDescription{
	{
		DeviceName:    "MicroEdge",
		Model:         schema.DeviceModelMicroEdgeV1,
		CheckLength:   CheckPayloadLengthME,
		Decode:        DecodeME,
		GetPointNames: GetMePointNames,
	},
	{
		DeviceName:    "MicroEdge",
		Model:         schema.DeviceModelMicroEdgeV2,
		CheckLength:   CheckPayloadLengthME,
		Decode:        DecodeME,
		GetPointNames: GetMePointNames,
	},
	{
		DeviceName:    "Droplet",
		Model:         schema.DeviceModelTHLM,
		CheckLength:   CheckPayloadLengthDroplet,
		Decode:        DecodeDropletTHLM,
		GetPointNames: GetTHLMPointNames,
	},
	{
		DeviceName:    "Droplet",
		Model:         schema.DeviceModelTHL,
		CheckLength:   CheckPayloadLengthDroplet,
		Decode:        DecodeDropletTHL,
		GetPointNames: GetTHLPointNames,
	},
	{
		DeviceName:    "Droplet",
		Model:         schema.DeviceModelTH,
		CheckLength:   CheckPayloadLengthDroplet,
		Decode:        DecodeDropletTH,
		GetPointNames: GetTHPointNames,
	},
	{
		DeviceName:    "ZipHydroTap",
		Model:         schema.DeviceModelZiptHydroTap,
		CheckLength:   CheckPayloadLengthZHT,
		Decode:        DecodeZHT,
		GetPointNames: GetZHTPointNames,
	},
	{
		DeviceName:    "Rubix",
		Model:         schema.DeviceModelRubix,
		CheckLength:   CheckPayloadLengthRubix,
		Decode:        DecodeRubix,
		GetPointNames: GetRubixPointNames,
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

func GetDevicePointNames(device *model.Device) []string {
	return GetDeviceDescription(device).GetPointNames()
}
