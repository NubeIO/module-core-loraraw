package utils

import (
	"crypto/rand"
	"errors"
	"reflect"
	"strings"

	"github.com/NubeIO/lib-utils-go/boolean"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/datatype"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
	log "github.com/sirupsen/logrus"
)

/*
 * Data Structure:
 * ------------------------------------------------------------------------------------------------------------------------------------------------------------------------
 * | 4 bytes address | 1 byte opts  | 1 byte nonce  | 1 byte length | Payload           | 4 bytes CMAC              | 1 bytes RSSI              |   1 bytes SNR           |
 * ------------------------------------------------------------------------------------------------------------------------------------------------------------------------
 * | data[0:3]       | data[4]      | data[5]       | data[6]       | data[7:dataLen-6] | data[dataLen-6:dataLen-2] | data[dataLen-2:dataLen-1] | data[dataLen-1:dataLen] |
 * ------------------------------------------------------------------------------------------------------------------------------------------------------------------------
 */
const (
	LORARAW_VERSION          = 0xC0
	LORARAW_VERSION_POSITION = 1
	LORARAW_OPTS_POSITION    = 4
	LORARAW_NONCE_POSITION   = 5
	LORARAW_LENGTH_POSITION  = 6
	LORARAW_PAYLOAD_START    = 7
	LORARAW_HEADER_LEN       = 4
)

type LoRaRAWOpts int

const (
	LORARAW_OPTS_UNCONFIRMED_UPLINK LoRaRAWOpts = 0
	LORARAW_OPTS_CONFIRMED_UPLINK   LoRaRAWOpts = 1
	LORARAW_OPTS_ACK                LoRaRAWOpts = 2
	LORARAW_OPTS_REQUEST            LoRaRAWOpts = 3
	LORARAW_OPTS_RESPONSE           LoRaRAWOpts = 4
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

func CheckLoRaRAWPayloadLength(data []byte) bool {
	// 4 bytes address | 1 byte opts | 1 byte nonce | 1 byte length | 1 byte rssi | 1 byte snr
	// NOTE CMAC is not sent when it's already decrypted by the driver
	payloadLength := len(data)
	payloadLength -= 9
	innerDataLength := GetLoRaRAWInnerPayloadLength(data)

	// inner data length must be <= encrypted payload length due to AES padding for payload to be mod16
	return innerDataLength <= payloadLength
}

func StripLoRaRAWPayload(data []byte) []byte {
	return data[LORARAW_PAYLOAD_START : LORARAW_PAYLOAD_START+GetLoRaRAWInnerPayloadLength(data)]
}

func GetLoRaRAWInnerPayloadLength(data []byte) int {
	return int(data[LORARAW_LENGTH_POSITION])
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

func GenerateRandomId() uint8 {
	// Create a new Rand instance with a seed
	var b [1]byte
	_, err := rand.Read(b[:])
	if err != nil {
		log.Errorf("error generating random id: %s", err.Error())
	}
	return b[0]
}
