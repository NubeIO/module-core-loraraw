package codec

import (
	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"

	"errors"
	"strings"
)

type UpdateDevicePointFunc func(name string, value float64, device *model.Device) error
type UpdateDevicePointErrorFunc func(name string, err error, device *model.Device) error
type UpdateDeviceWrittenPointFunc func(name string, value float64, messageId uint8, device *model.Device) error
type UpdateDeviceWrittenPointErrorFunc func(name string, err error, messageId uint8, device *model.Device) error
type UpdateDeviceMetaTagsFunc func(uuid string, metaTags []*model.DeviceMetaTag) error

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
		updateDevPntErrFnc UpdateDevicePointErrorFunc,
		updateDevMetaTagsFnc UpdateDeviceMetaTagsFunc,
	) error
	DecodeResponse func(
		dataHex string,
		payloadBytes []byte,
		devDesc *LoRaDeviceDescription,
		device *model.Device,
		updateDevPntFnc UpdateDeviceWrittenPointFunc,
		updateDevPntErrFnc UpdateDeviceWrittenPointErrorFunc,
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
	_ UpdateDevicePointErrorFunc,
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
	_ UpdateDeviceWrittenPointErrorFunc,
	_ UpdateDeviceMetaTagsFunc,
) error {
	return errors.New("nil decode function called")
}

func NilLoRaDeviceDescriptionGetPointsStruct() []string {
	return []string{}
}

func GetDeviceDescription(device *model.Device, deviceDescriptions []LoRaDeviceDescription) *LoRaDeviceDescription {
	for _, devDesc := range deviceDescriptions {
		if strings.EqualFold(device.Model, devDesc.Model) {
			return &devDesc
		}
	}
	return &NilLoRaDeviceDescription
}

func GetDevicePointNames(device *model.Device, deviceDescriptions []LoRaDeviceDescription) []string {
	return GetDeviceDescription(device, deviceDescriptions).GetPointNames()
}
