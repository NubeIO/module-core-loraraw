package rubixDataEncoding

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"unsafe"

	"github.com/NubeIO/module-core-loraraw/codec"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
	log "github.com/sirupsen/logrus"
)

func canDecode(serialData *SerialData) bool {
	return serialData.ReadBitPos < (len(serialData.Buffer)*8 - DATA_TYPE_BIT_COUNT)
}

func parseMetaData(serialData *SerialData) (MetaDataKey, PositionData) {
	var positionData PositionData
	if hasPositionalData(serialData) {
		positionVector, shiftPos, bytesRequired := getVector(serialData, 8, serialData.ReadBitPos)
		positionByte := uint8(vectorToBits(positionVector, 8, shiftPos, bytesRequired))
		positionData = parsePosition(positionByte)
	}
	typeVector, shiftPos, bytesRequired := getVector(serialData, DATA_TYPE_BIT_COUNT, serialData.ReadBitPos)
	return MetaDataKey(vectorToBits(typeVector, DATA_TYPE_BIT_COUNT, shiftPos, bytesRequired)), positionData
}

func parsePosition(positionByte uint8) PositionData {
	// first 5 bits are are the ID, last 3 are the point type
	return PositionData{
		ID:   int(positionByte & 0x1F),
		Type: PositionDataType(positionByte >> 5),
	}
}

func vectorToBits(dataVector []byte, bitCount, shiftPos, bytesRequired int) uint64 {
	data64 := uint64(0)
	shiftLeft := bytesRequired*8 - bitCount - shiftPos
	for i := 0; i < len(dataVector)-1; i++ {
		data64 |= uint64(dataVector[i])
		if i != len(dataVector)-2 {
			data64 <<= 8
		}
	}
	// Shift forward and add last bits
	data64 <<= 8 - shiftLeft
	data64 |= uint64(dataVector[len(dataVector)-1]) >> shiftLeft

	// Mask out leading bits
	if bitCount < 64 {
		data64 &= (1 << uint(bitCount)) - 1
	}

	return data64
}

func getBitCount(low, high int, decimal int) int {
	// Calculate the largest possible number
	requiredSize := uint64(high-low) * uint64(math.Pow(10, float64(decimal)))

	// Calculate number of bits required to fit number
	bitCount := 1
	for math.Pow(2, float64(bitCount)) < float64(requiredSize) {
		bitCount++
	}

	return bitCount
}

func getVector(serialData *SerialData, bitCount, pos int) (dataVector []byte, shiftPos, bytesRequired int) {
	// Calculate shift required from positioning of data
	shiftFromPos := pos % 8
	// Total bytes required for all data, can contain unwanted leading and trailing data
	bytesRequired = bitCount / 8
	if bitCount%8 != 0 {
		bytesRequired++
	}
	// Checks if an extra byte is required
	if bitCount+shiftFromPos > bytesRequired*8 {
		bytesRequired++
	}
	// Build vector from relative bytes
	byteLocation := pos / 8
	dataVector = make([]byte, bytesRequired)
	if byteLocation+bytesRequired > len(serialData.Buffer) {
		dataVector[0] = 0 // Handle the case where requested data is beyond buffer size
	} else {
		for i := 0; i < bytesRequired; i++ {
			dataVector[i] = serialData.Buffer[byteLocation+i]
		}
	}
	// Update bit read position
	serialData.ReadBitPos += bitCount

	// Assign shiftPos and bytesRequired to the variables pointed by the pointers
	shiftPos = shiftFromPos

	return dataVector, shiftPos, bytesRequired
}

func decodeData(serialData *SerialData, metaDataKey MetaDataKey, data interface{}) error {
	if metaDataKey == 0 {
		return errors.New("invalid MetaDataKey")
	}
	metaData := getMetaData(metaDataKey)
	var shiftPos, bytesRequired, bitCount int
	var dataBits BIT_TYPE
	var dataVector []byte

	switch metaData.dataType {
	case FIXEDPOINT:
		bitCount = getBitCount(metaData.lowValue, metaData.highValue, metaData.decimalPoint)
		dataVector, shiftPos, bytesRequired = getVector(serialData, bitCount, serialData.ReadBitPos)
		dataBits = BIT_TYPE(vectorToBits(dataVector, bitCount, shiftPos, bytesRequired))
		switch v := data.(type) {
		case *float32:
			*v = float32(float64(dataBits)/math.Pow(10, float64(metaData.decimalPoint)) + float64(metaData.lowValue))
		default:
			return fmt.Errorf("unsupported data type: %T", data)
		}
	case DATAPOINT:
		bitCount = metaData.byteCount * 8
		dataVector, shiftPos, bytesRequired = getVector(serialData, bitCount, serialData.ReadBitPos)
		dataBits = BIT_TYPE(vectorToBits(dataVector, bitCount, shiftPos, bytesRequired))
		switch metaDataKey {
		case MDK_CHAR:
			if v, ok := data.(*byte); ok {
				*v = byte(dataBits)
			} else {
				return fmt.Errorf("invalid type for MDK_CHAR: %T", data)
			}
		case MDK_UINT_8:
			if v, ok := data.(*uint8); ok {
				*v = uint8(dataBits)
			} else {
				return fmt.Errorf("invalid type for MDK_UINT_8: %T", data)
			}
		case MDK_INT_8:
			if v, ok := data.(*int8); ok {
				*v = int8(dataBits)
			} else {
				return fmt.Errorf("invalid type for MDK_INT_8: %T", data)
			}
		case MDK_UINT_16:
			if v, ok := data.(*uint16); ok {
				*v = uint16(dataBits)
			} else {
				return fmt.Errorf("invalid type for MDK_UINT_16: %T", data)
			}

		case MDK_INT_16:
			if v, ok := data.(*int16); ok {
				*v = int16(dataBits)
			} else {
				return fmt.Errorf("invalid type for MDK_INT_16: %T", data)
			}
		case MDK_UINT_32:
			if v, ok := data.(*uint32); ok {
				*v = uint32(dataBits)
			} else {
				return fmt.Errorf("invalid type for MDK_UINT_32: %T", data)
			}
		case MDK_INT_32:
			if v, ok := data.(*int32); ok {
				*v = int32(dataBits)
			} else {
				return fmt.Errorf("invalid type for MDK_INT_32: %T", data)
			}
		case MDK_UINT_64:
			if v, ok := data.(*uint64); ok {
				*v = uint64(dataBits)
			} else {
				return fmt.Errorf("invalid type for MDK_UINT_64: %T", data)
			}
		case MDK_INT_64:
			if v, ok := data.(*int64); ok {
				*v = int64(dataBits)
			} else {
				return fmt.Errorf("invalid type for MDK_INT_64: %T", data)
			}
		case MDK_FLOAT:
			if v, ok := data.(*float32); ok {
				floatValue := *(*float32)(unsafe.Pointer(&dataBits))
				*v = floatValue
			} else {
				return fmt.Errorf("invalid type for MDK_FLOAT: %T", data)
			}
		case MDK_ERROR:
			if v, ok := data.(*uint8); ok {
				*v = uint8(dataBits)
			} else {
				return fmt.Errorf("invalid type for MDK_ERROR: %T", data)
			}
		default:
			return fmt.Errorf("unsupported MetaDataKey: %v", metaDataKey)
		}
	default:
		return fmt.Errorf("unsupported data type1: %v", metaData.dataType)
	}
	return nil
}

func generateFieldName(metaDataKey MetaDataKey, pos PositionData) string {
	id := pos.ID + 1
	switch pos.Type {
	case PositionDataType_GENERAL:
		return metaDataKey.String() + "-" + strconv.Itoa(id)
	case PositionDataType_UO:
		return "UO-" + strconv.Itoa(id)
	case PositionDataType_DO:
		return "DO-" + strconv.Itoa(id)
	case PositionDataType_UI:
		return "UI-" + strconv.Itoa(id)
	case PositionDataType_DI:
		return "DI-" + strconv.Itoa(id)
	case PositionDataType_UVP:
		return "UVP-" + strconv.Itoa(id)
	case PositionDataType_UVP2:
		return "UVP-" + strconv.Itoa(id+32)
	case PositionDataType_DVP:
		return "DVP-" + strconv.Itoa(id)
	default:
		return "INVALID_POSITION_TYPE-" + strconv.Itoa(int(pos.Type)) + "-" + strconv.Itoa(id)
	}
}

func DecodeRubixUplink(
	_ string,
	payloadBytes []byte,
	_ *codec.LoRaDeviceDescription,
	device *model.Device,
	updatePointFn codec.UpdateDevicePointFunc,
	updatePointErrFn codec.UpdateDevicePointErrorFunc,
	_ codec.UpdateDeviceMetaTagsFunc,
) error {
	return DecodeRubix(payloadBytes, device, updatePointFn, updatePointErrFn, nil, nil)
}

func DecodeRubixResponse(
	_ string,
	payloadBytes []byte,
	_ *codec.LoRaDeviceDescription,
	device *model.Device,
	updateWrittenPointFn codec.UpdateDeviceWrittenPointFunc,
	updateWrittenPointErrFn codec.UpdateDeviceWrittenPointErrorFunc,
	_ codec.UpdateDeviceMetaTagsFunc,
) error {
	return DecodeRubix(payloadBytes, device, nil, nil, updateWrittenPointFn, updateWrittenPointErrFn)
}

func DecodeRubix(
	payloadBytes []byte,
	device *model.Device,
	updatePointFn codec.UpdateDevicePointFunc,
	updatePointErrFn codec.UpdateDevicePointErrorFunc,
	updateWrittenPointFn codec.UpdateDeviceWrittenPointFunc,
	updateWrittenPointErrFn codec.UpdateDeviceWrittenPointErrorFunc,
) error {
	serialData := NewSerialDataWithBuffer(payloadBytes)

	hasPos := hasPositionalData(serialData)

	positionData := PositionData{
		ID:   0,
		Type: PositionDataType_GENERAL,
	}
	for canDecode(serialData) {
		metaDataKey, positionDataNew := parseMetaData(serialData)
		if hasPos {
			positionData = positionDataNew
		}
		name, value, err := decodePointRubix(serialData, metaDataKey, hasPos, positionData, device, updatePointFn)
		if updatePointFn != nil {
			if err != nil {
				updatePointErrFn(name, err, device)
			} else {
				updatePointFn(name, value, device)
			}
		} else {
			if err != nil {
				updateWrittenPointErrFn(name, err, 0, device)
			} else {
				updateWrittenPointFn(name, value, 0, device)
			}
		}

		positionData.ID++ // might be overwritten anyway if hasPos is true
	}

	return nil
}

func decodePointRubix(serialData *SerialData, metaDataKey MetaDataKey, hasPos bool, position PositionData, device *model.Device, updatePointFn codec.UpdateDevicePointFunc) (name string, value float64, err error) {
	var (
		f32  float32
		u8   uint8
		i8   int8
		u16  uint16
		i16  int16
		u32  uint32
		i32  int32
		u64  uint64
		i64  int64
		char byte
	)

	name = generateFieldName(metaDataKey, position)

	switch metaDataKey {
	case MDK_TEMP:
		fallthrough
	case MDK_RH:
		fallthrough
	case MDK_LUX:
		fallthrough
	case MDK_MOVEMENT:
		fallthrough
	case MDK_COUNTER:
		fallthrough
	case MDK_DIGITAL:
		fallthrough
	case MDK_VOLTAGE_0_10:
		fallthrough
	case MDK_MILLIAMPS_4_20:
		fallthrough
	case MDK_OHM:
		fallthrough
	case MDK_CO2:
		fallthrough
	case MDK_BATTERY_VOLTAGE:
		fallthrough
	case MDK_PUSH_FREQUENCY:
		fallthrough
	case MDK_ANALOG_IN:
		fallthrough
	case MDK_FIRMWARE_VERSION:
		fallthrough
	case MDK_HARDWARE_VERSION:
		decodeData(serialData, metaDataKey, &f32)
		value = float64(f32)

	case MDK_UINT_8:
		decodeData(serialData, metaDataKey, &u8)
		value = float64(u8)
	case MDK_INT_8:
		decodeData(serialData, metaDataKey, &i8)
		value = float64(i8)
	case MDK_UINT_16:
		decodeData(serialData, metaDataKey, &u16)
		value = float64(u16)
	case MDK_INT_16:
		decodeData(serialData, metaDataKey, &i16)
		value = float64(i16)
	case MDK_UINT_32:
		decodeData(serialData, metaDataKey, &u32)
		value = float64(u32)
	case MDK_INT_32:
		decodeData(serialData, metaDataKey, &i32)
		value = float64(i32)
	case MDK_UINT_64:
		decodeData(serialData, metaDataKey, &u64)
		value = float64(u64)
	case MDK_INT_64:
		decodeData(serialData, metaDataKey, &i64)
		value = float64(i64)
	case MDK_CHAR:
		decodeData(serialData, metaDataKey, &char)
		value = float64(char)
	case MDK_FLOAT:
		decodeData(serialData, metaDataKey, &f32)
		value = float64(f32)
	case MDK_BOOL:
		decodeData(serialData, metaDataKey, &f32)
		value = float64(f32)
	case MDK_ERROR:
		var errCode uint8 = 0
		decodeData(serialData, metaDataKey, &errCode)
		if errCode != 0 {
			// TODO: add error code to string conversion for known common errors
			return name, 0, errors.New("RDE error: " + strconv.Itoa(int(errCode)))
		}
		value = 0
	case 0:
		log.Debug("reached end of data with some bits left over")
	default:
		log.Errorf("Unknown MetaDataKey: %d", metaDataKey)
	}

	return name, value, nil
}

func GetRubixPointNames() []string {
	return codec.GetCommonValueNames()
}

func CheckPayloadLengthRubix(_ string) bool {
	return true
}
