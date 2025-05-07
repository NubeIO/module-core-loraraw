package endec

import (
	"github.com/NubeIO/module-core-loraraw/schema"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"

	"errors"
	"strings"
)

type UpdateDevicePointFunc func(name string, value float64, device *model.Device) error
type UpdateDeviceWrittenPointFunc func(name string, value float64, err error, messageId uint8, device *model.Device) error
type UpdateDeviceMetaTagsFunc func(uuid string, metaTags []*model.DeviceMetaTag) error

type SendAckToDeviceFunc func(device *model.Device, messageId uint8) error
type InternalPointUpdate func(point *model.Point) (*model.Point, error)
type DequeuePointWriteFunc func(messageId uint8) *model.Point

type LoRaDeviceDescription struct {
	DeviceName   string
	Model        string
	SensorCode   string
	CheckLength  func(data string) bool
	DecodeUplink func(
		dataHex string,
		payloadBytes []byte,
		devDesc *LoRaDeviceDescription,
		device *model.Device,
		updateDevPntFnc UpdateDevicePointFunc,
		updateDevMetaTagsFnc UpdateDeviceMetaTagsFunc,
	) error
	DecodeResponse func(
		dataHex string,
		payloadBytes []byte,
		devDesc *LoRaDeviceDescription,
		device *model.Device,
		updateDevPntFnc UpdateDeviceWrittenPointFunc,
		updateDevMetaTagsFnc UpdateDeviceMetaTagsFunc,
	) error
	GetPointNames func() []string
	IsLoRaRAW     bool
}

var NilLoRaDeviceDescription = LoRaDeviceDescription{
	DeviceName:     "",
	Model:          "",
	SensorCode:     "",
	CheckLength:    NilLoRaDeviceDescriptionCheckLength,
	DecodeUplink:   NilLoRaDeviceDescriptionDecode,
	DecodeResponse: NilLoRaDeviceDescriptionDecodeResponse,
	GetPointNames:  NilLoRaDeviceDescriptionGetPointsStruct,
}

func NilLoRaDeviceDescriptionCheckLength(data string) bool {
	return false
}

func NilLoRaDeviceDescriptionDecode(
	_ string,
	_ []byte,
	_ *LoRaDeviceDescription,
	_ *model.Device,
	_ UpdateDevicePointFunc,
	_ UpdateDeviceMetaTagsFunc,
) error {
	return errors.New("nil decode function called")
}

func NilLoRaDeviceDescriptionDecodeResponse(
	_ string,
	_ []byte,
	_ *LoRaDeviceDescription,
	_ *model.Device,
	_ UpdateDeviceWrittenPointFunc,
	_ UpdateDeviceMetaTagsFunc,
) error {
	return errors.New("nil decode function called")
}

func NilLoRaDeviceDescriptionGetPointsStruct() []string {
	return []string{}
}

var LoRaDeviceDescriptions = [...]LoRaDeviceDescription{
	{
		// LEGACY DEVICE. PLS REMOVE IN FUTURE
		DeviceName:    "MicroEdge",
		Model:         schema.DeviceModelMicroEdgeV1,
		CheckLength:   CheckPayloadLengthME,
		DecodeUplink:  DecodeME,
		GetPointNames: GetMePointNames,
		IsLoRaRAW:     false,
	},
	{
		// LEGACY DEVICE. PLS REMOVE IN FUTURE
		DeviceName:    "MicroEdge",
		Model:         schema.DeviceModelMicroEdgeV2,
		CheckLength:   CheckPayloadLengthME,
		DecodeUplink:  DecodeME,
		GetPointNames: GetMePointNames,
		IsLoRaRAW:     false,
	},
	{
		// LEGACY DEVICE. PLS REMOVE IN FUTURE
		DeviceName:    "Droplet",
		Model:         schema.DeviceModelTHLM,
		CheckLength:   CheckPayloadLengthDroplet,
		DecodeUplink:  DecodeDropletTHLM,
		GetPointNames: GetTHLMPointNames,
		IsLoRaRAW:     false,
	},
	{
		// LEGACY DEVICE. PLS REMOVE IN FUTURE
		DeviceName:    "Droplet",
		Model:         schema.DeviceModelTHL,
		CheckLength:   CheckPayloadLengthDroplet,
		DecodeUplink:  DecodeDropletTHL,
		GetPointNames: GetTHLPointNames,
		IsLoRaRAW:     false,
	},
	{
		// LEGACY DEVICE. PLS REMOVE IN FUTURE
		DeviceName:    "Droplet",
		Model:         schema.DeviceModelTH,
		CheckLength:   CheckPayloadLengthDroplet,
		DecodeUplink:  DecodeDropletTH,
		GetPointNames: GetTHPointNames,
		IsLoRaRAW:     false,
	},
	{
		// LEGACY DEVICE. PLS REMOVE IN FUTURE
		DeviceName:    "ZipHydroTap",
		Model:         schema.DeviceModelZiptHydroTap,
		CheckLength:   CheckPayloadLengthZHT,
		DecodeUplink:  DecodeZHT,
		GetPointNames: GetZHTPointNames,
		IsLoRaRAW:     true,
	},
	{
		DeviceName:     "Rubix",
		Model:          schema.DeviceModelRubix,
		CheckLength:    CheckPayloadLengthRubix,
		DecodeUplink:   DecodeRubixUplink,
		DecodeResponse: DecodeRubixResponse,
		GetPointNames:  GetRubixPointNames,
		IsLoRaRAW:      true,
	},
}

func GetDeviceDescription(device *model.Device) *LoRaDeviceDescription {
	for _, devDesc := range LoRaDeviceDescriptions {
		if strings.EqualFold(device.Model, devDesc.Model) {
			return &devDesc
		}
	}
	return &NilLoRaDeviceDescription
}

func GetDevicePointNames(device *model.Device) []string {
	return GetDeviceDescription(device).GetPointNames()
}
