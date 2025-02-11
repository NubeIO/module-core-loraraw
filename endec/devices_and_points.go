package endec

import (
	"errors"
	"github.com/NubeIO/module-core-loraraw/schema"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"

	"strings"
)

type UpdateDevicePointFunc func(name string, value float64, device *model.Device) error
type UpdateDeviceMetaTagsFunc func(uuid string, metaTags []*model.DeviceMetaTag) error
type DequeuePointWriteFunc func(messageId uint8) *model.Point
type InternalPointUpdate func(point *model.Point) (*model.Point, error)

type LoRaDeviceDescription struct {
	DeviceName  string
	Model       string
	SensorCode  string
	CheckLength func(data string) bool
	Decode      func(
		data string,
		devDesc *LoRaDeviceDescription,
		device *model.Device,
		updateDevPntFnc UpdateDevicePointFunc,
		updateDevMetaTagsFnc UpdateDeviceMetaTagsFunc,
		dequeuePointWriteFunc DequeuePointWriteFunc,
		internalPointWriteFunc InternalPointUpdate,
	) error
	GetPointNames func() []string
	IsLoRaRAW     bool
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

func NilLoRaDeviceDescriptionDecode(
	_ string,
	_ *LoRaDeviceDescription,
	_ *model.Device,
	_ UpdateDevicePointFunc,
	_ UpdateDeviceMetaTagsFunc,
	_ DequeuePointWriteFunc,
	_ InternalPointUpdate,
) error {
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
		IsLoRaRAW:     false,
	},
	{
		DeviceName:    "MicroEdge",
		Model:         schema.DeviceModelMicroEdgeV2,
		CheckLength:   CheckPayloadLengthME,
		Decode:        DecodeME,
		GetPointNames: GetMePointNames,
		IsLoRaRAW:     false,
	},
	{
		DeviceName:    "Droplet",
		Model:         schema.DeviceModelTHLM,
		CheckLength:   CheckPayloadLengthDroplet,
		Decode:        DecodeDropletTHLM,
		GetPointNames: GetTHLMPointNames,
		IsLoRaRAW:     false,
	},
	{
		DeviceName:    "Droplet",
		Model:         schema.DeviceModelTHL,
		CheckLength:   CheckPayloadLengthDroplet,
		Decode:        DecodeDropletTHL,
		GetPointNames: GetTHLPointNames,
		IsLoRaRAW:     false,
	},
	{
		DeviceName:    "Droplet",
		Model:         schema.DeviceModelTH,
		CheckLength:   CheckPayloadLengthDroplet,
		Decode:        DecodeDropletTH,
		GetPointNames: GetTHPointNames,
		IsLoRaRAW:     false,
	},
	{
		DeviceName:    "ZipHydroTap",
		Model:         schema.DeviceModelZiptHydroTap,
		CheckLength:   CheckPayloadLengthZHT,
		Decode:        DecodeZHT,
		GetPointNames: GetZHTPointNames,
		IsLoRaRAW:     true,
	},
	{
		DeviceName:    "Rubix",
		Model:         schema.DeviceModelRubix,
		CheckLength:   CheckPayloadLengthRubix,
		Decode:        DecodeRubix,
		GetPointNames: GetRubixPointNames,
		IsLoRaRAW:     true,
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
