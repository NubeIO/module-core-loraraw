package endec

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strconv"
	"unsafe"

	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
	log "github.com/sirupsen/logrus"
)

const (
	TempField           = "temp"
	RHField             = "rh"
	LuxField            = "lux"
	MovementField       = "movement"
	CounterField        = "count"
	DigitalField        = "digital"
	VoltageField        = "0-10v"
	MilliampsField      = "4-20ma"
	OhmField            = "ohm"
	CO2Field            = "co2"
	BatteryVoltageField = "battery-voltage"
	PushFrequencyField  = "push-frequency"
	RawField            = "raw"
	UOField             = "uo"
	UIField             = "ui"
	DOField             = "do"
	DIField             = "di"
	FwVersionField      = "firmware-version"
	HwVersionField      = "hardware-version"
	BoolField           = "bool"
	CharField           = "char"
	UInt8Field          = "uint_8"
	Int8Field           = "int_8"
	UInt16Field         = "uint_16"
	Int16Field          = "int_16"
	UInt32Field         = "uint_32"
	Int32Field          = "int_32"
	UInt64Field         = "uint_64"
	Int64Field          = "int_64"
	FloatField          = "float"
	DoubleField         = "double"
	ErrorField          = "error"
)

func canDecode(serialData *SerialData) bool {
	return serialData.ReadBitPos < (len(serialData.Buffer)*8 - HEADER_BIT_COUNT)
}

func getHeader(serialData *SerialData, position *uint8) MetaDataKey {
	if HasPositionalData(serialData) {
		positionVector, shiftPos, bytesRequired := getVector(serialData, 8, serialData.ReadBitPos)
		*position = uint8(vectorToBits(positionVector, 8, shiftPos, bytesRequired))
	}
	typeVector, shiftPos, bytesRequired := getVector(serialData, HEADER_BIT_COUNT, serialData.ReadBitPos)
	return MetaDataKey(vectorToBits(typeVector, HEADER_BIT_COUNT, shiftPos, bytesRequired))
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

func decodeData(serialData *SerialData, header MetaDataKey, data interface{}) error {
	if header == 0 {
		return errors.New("invalid header")
	}
	metaData := getMetaData(header)
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
		switch header {
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
		default:
			return fmt.Errorf("unsupported header: %v", header)
		}
	default:
		return fmt.Errorf("unsupported data type1: %v", metaData.dataType)
	}
	return nil
}

func generateFieldName(baseName string, hasPosition bool, pos *uint8) string {
	if !hasPosition {
		*pos++
	}
	return baseName + "-" + strconv.Itoa(int(*pos))
}

func DecodeRubix(
	data string,
	_ *LoRaDeviceDescription,
	device *model.Device,
	updatePointFn UpdateDevicePointFunc,
	_ UpdateDeviceMetaTagsFunc,
	dequeuePointWriteFunc DequeuePointWriteFunc,
	internalPointUpdate InternalPointUpdate,
) error {
	var (
		temperature float32
		rh          float32
		lux         float32
		movement    float32
		counter     float32
		digital     float32
		voltage     float32
		amplitude   float32
		ohm         float32
		co2         float32
		batVol      float32
		pushFreq    float32
		raw         float32
		uo          float32
		ui          float32
		do          float32
		di          float32
		fwVer       float32
		hwVer       float32
		u8          uint8
		i8          int8
		u16         uint16
		i16         int16
		u32         uint32
		i32         int32
		u64         uint64
		i64         int64
		char        byte
		fl1         float32
		b1          float32
		error       uint16
	)

	dataBytes, err := hex.DecodeString(data)
	if err != nil {
		log.Errorf("Error decoding hex string: %v", err)
		return err
	}

	serialData := NewSerialDataWithBuffer(dataBytes)

	hasPos := HasPositionalData(serialData)
	var position uint8 = 0
	if HasRequestData(serialData) || HasResponseData(serialData) {
		messageId := GetMessageId(serialData)
		point := dequeuePointWriteFunc(messageId)
		if point != nil {
			// TODO: May be re-add to the queue if internal point update fails???
			_, _ = internalPointUpdate(point)
		}
		UpdateBitPositionsForHeaderByte(serialData)
	}

	for canDecode(serialData) {
		header := getHeader(serialData, &position)
		switch header {
		case MDK_TEMP:
			decodeData(serialData, header, &temperature)
			_ = updatePointFn(generateFieldName(TempField, hasPos, &position), float64(temperature), device)
		case MDK_RH:
			decodeData(serialData, header, &rh)
			_ = updatePointFn(generateFieldName(RHField, hasPos, &position), float64(rh), device)
		case MDK_LUX:
			decodeData(serialData, header, &lux)
			_ = updatePointFn(generateFieldName(LuxField, hasPos, &position), float64(lux), device)
		case MDK_MOVEMENT:
			decodeData(serialData, header, &movement)
			_ = updatePointFn(generateFieldName(MovementField, hasPos, &position), float64(movement), device)
		case MDK_COUNTER:
			decodeData(serialData, header, &counter)
			_ = updatePointFn(generateFieldName(CounterField, hasPos, &position), float64(counter), device)
		case MDK_DIGITAL:
			decodeData(serialData, header, &digital)
			_ = updatePointFn(generateFieldName(DigitalField, hasPos, &position), float64(digital), device)
		case MDK_VOLTAGE_0_10:
			decodeData(serialData, header, &voltage)
			_ = updatePointFn(generateFieldName(VoltageField, hasPos, &position), float64(voltage), device)
		case MDK_MILLIAMPS_4_20:
			decodeData(serialData, header, &amplitude)
			_ = updatePointFn(generateFieldName(MilliampsField, hasPos, &position), float64(amplitude), device)
		case MDK_OHM:
			decodeData(serialData, header, &ohm)
			_ = updatePointFn(generateFieldName(OhmField, hasPos, &position), float64(ohm), device)
		case MDK_CO2:
			decodeData(serialData, header, &co2)
			_ = updatePointFn(generateFieldName(CO2Field, hasPos, &position), float64(co2), device)
		case MDK_BATTERY_VOLTAGE:
			decodeData(serialData, header, &batVol)
			_ = updatePointFn(generateFieldName(BatteryVoltageField, hasPos, &position), float64(batVol), device)
		case MDK_PUSH_FREQUENCY:
			decodeData(serialData, header, &pushFreq)
			_ = updatePointFn(generateFieldName(PushFrequencyField, hasPos, &position), float64(pushFreq), device)
		case MDK_RAW:
			decodeData(serialData, header, &raw)
			_ = updatePointFn(generateFieldName(RawField, hasPos, &position), float64(raw), device)
		case MDK_UO:
			decodeData(serialData, header, &uo)
			_ = updatePointFn(generateFieldName(UOField, hasPos, &position), float64(uo), device)
		case MDK_UI:
			decodeData(serialData, header, &ui)
			_ = updatePointFn(generateFieldName(UIField, hasPos, &position), float64(ui), device)
		case MDK_DO:
			decodeData(serialData, header, &do)
			_ = updatePointFn(generateFieldName(DOField, hasPos, &position), float64(do), device)
		case MDK_DI:
			decodeData(serialData, header, &di)
			_ = updatePointFn(generateFieldName(DIField, hasPos, &position), float64(di), device)
		case MDK_FIRMWARE_VERSION:
			decodeData(serialData, header, &fwVer)
			_ = updatePointFn(generateFieldName(FwVersionField, hasPos, &position), float64(fwVer), device)
		case MDK_HARDWARE_VERSION:
			decodeData(serialData, header, &hwVer)
			_ = updatePointFn(generateFieldName(HwVersionField, hasPos, &position), float64(hwVer), device)
		case MDK_UINT_8:
			decodeData(serialData, header, &u8)
			_ = updatePointFn(generateFieldName(UInt8Field, hasPos, &position), float64(u8), device)
		case MDK_INT_8:
			decodeData(serialData, header, &i8)
			_ = updatePointFn(generateFieldName(Int8Field, hasPos, &position), float64(i8), device)
		case MDK_UINT_16:
			decodeData(serialData, header, &u16)
			_ = updatePointFn(generateFieldName(UInt16Field, hasPos, &position), float64(u16), device)
		case MDK_INT_16:
			decodeData(serialData, header, &i16)
			_ = updatePointFn(generateFieldName(Int16Field, hasPos, &position), float64(i16), device)
		case MDK_UINT_32:
			decodeData(serialData, header, &u32)
			_ = updatePointFn(generateFieldName(UInt32Field, hasPos, &position), float64(u32), device)
		case MDK_INT_32:
			decodeData(serialData, header, &i32)
			_ = updatePointFn(generateFieldName(Int32Field, hasPos, &position), float64(i32), device)
		case MDK_UINT_64:
			decodeData(serialData, header, &u64)
			_ = updatePointFn(generateFieldName(UInt64Field, hasPos, &position), float64(u64), device)
		case MDK_INT_64:
			decodeData(serialData, header, &i64)
			_ = updatePointFn(generateFieldName(Int64Field, hasPos, &position), float64(i64), device)
		case MDK_CHAR:
			decodeData(serialData, header, &char)
			_ = updatePointFn(generateFieldName(CharField, hasPos, &position), float64(char), device)
		case MDK_FLOAT:
			decodeData(serialData, header, &fl1)
			_ = updatePointFn(generateFieldName(FloatField, hasPos, &position), float64(fl1), device)
		case MDK_BOOL:
			decodeData(serialData, header, &b1)
			_ = updatePointFn(generateFieldName(BoolField, hasPos, &position), float64(b1), device)
		case MDK_ERROR:
			decodeData(serialData, header, &error)
			_ = updatePointFn(generateFieldName(ErrorField, hasPos, &position), float64(error), device)
		case 0:
			log.Debug("reached end of data with some bits left over")
		default:
			log.Errorf("Unknown header: %d", header)
		}
	}
	return nil
}

func GetRubixPointNames() []string {
	return GetCommonValueNames()
}

func CheckPayloadLengthRubix(_ string) bool {
	return true
}
