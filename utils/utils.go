package utils

import (
	"errors"
	"github.com/NubeIO/lib-utils-go/boolean"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/datatype"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
	"reflect"
	"strconv"
	"strings"
)

func BoolToFloat(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

func GetStructFieldJSONNameByName(thing interface{}, name string) string {
	field, err := reflect.TypeOf(thing).FieldByName(name)
	if !err {
		panic(err)
	}
	return GetReflectFieldJSONName(field)
}

func GetReflectFieldJSONName(field reflect.StructField) string {
	fieldName := field.Name

	switch jsonTag := field.Tag.Get("json"); jsonTag {
	case "-":
		fallthrough
	case "":
		return fieldName
	default:
		parts := strings.Split(jsonTag, ",")
		name := parts[0]
		if name == "" {
			name = fieldName
		}
		return name
	}
}

func CheckLoRaRAWPayloadLength(data string) bool {
	// 4 bytes address | 1 byte opts | 1 byte nonce | 1 byte length | 1 byte rssi | 1 byte snr
	// NOTE CMAC is not sent when it's already decrypted by the driver
	payloadLength := len(data) / 2
	payloadLength -= 9
	innerDataLength := GetLoRaRAWInnerPayloadLength(data)

	// inner data length must be <= encrypted payload length due to AES padding for payload to be mod16
	return innerDataLength <= payloadLength
}

func StripLoRaRAWPayload(data string) string {
	return data[14 : 14+(GetLoRaRAWInnerPayloadLength(data)*2)]
}

func GetLoRaRAWInnerPayloadLength(data string) int {
	dataLength, _ := strconv.ParseInt(data[12:14], 16, 0)
	return int(dataLength)
}

func IsWriteable(writeMode datatype.WriteMode) bool {
	switch writeMode {
	case datatype.ReadOnce, datatype.ReadOnly:
		return false
	case datatype.WriteOnce, datatype.WriteOnceReadOnce, datatype.WriteAlways, datatype.WriteOnceThenRead, datatype.WriteAndMaintain:
		return true
	default:
		return false
	}
}

func ResetWriteableProperties(point *model.Point) *model.Point {
	point.WriteValueOriginal = nil
	point.WriteValue = nil
	point.WritePriority = nil
	point.CurrentPriority = nil
	point.EnableWriteable = boolean.NewFalse()
	point.WritePollRequired = boolean.NewFalse()
	return point
}

func SafeDereferenceUint8(ptr *int) (uint8, error) {
	if ptr == nil {
		return 0, errors.New("attempting to dereference a nil uint8 pointer")
	}
	return uint8(*ptr), nil
}
