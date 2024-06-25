package decoder

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strconv"
	"unsafe"
)

const HEADER_BIT_COUNT = 6

type DataType int
type MetaDataKey int
type BIT_TYPE uint64

type MetaData struct {
	dataType     DataType
	lowValue     int
	highValue    int
	decimalPoint int
	byteCount    int
}

type TRubix struct {
	CommonValues
	Temp      float32 `json:"temp-1"`
	Rh        float32 `json:"rh-2"`
	Lux       float32 `json:"lux-3"`
	Movement  float32 `json:"movement-4"`
	Count     float32 `json:"count-5"`
	Digital   float32 `json:"digital-6"`
	Voltage   float32 `json:"0-10v-7"`
	Amplitude float32 `json:"4-20ma-8"`
	Ohm       float32 `json:"ohm-9"`
	Co2       float32 `json:"co2-10"`
	BatVol    float32 `json:"battery-voltage-11"`
	PushFreq  float32 `json:"push-frequency-12"`
	Raw       float32 `json:"raw-13"`
	Uo        float32 `json:"uo-14"`
	Ui        float32 `json:"ui-15"`
	Do        float32 `json:"do-16"`
	Di        float32 `json:"di-17"`
	FwVer     float32 `json:"firmware-version-18"`
	HwVer     float32 `json:"hardware-version-19"`
	Uint8     uint8   `json:"uint_8-20"`
	Int8      int8    `json:"int_8-21"`
	Uint16    uint16  `json:"uint_16-22"`
	Int16     int16   `json:"int_16-23"`
	Uint32    uint32  `json:"uint_32-24"`
	Int32     int32   `json:"int_32-25"`
	Uint64    uint64  `json:"uint_64-26"`
	Int64     int64   `json:"int_64-27"`
	Char      byte    `json:"char-28"`
	Float1    float32 `json:"float-29"`
	Float2    float32 `json:"float-30"`
	Float3    float32 `json:"float-31"`
}

const (
	MDK_TEMP             = 1
	MDK_RH               = 2
	MDK_LUX              = 3
	MDK_MOVEMENT         = 4
	MDK_COUNTER          = 5
	MDK_DIGITAL          = 6
	MDK_VOLTAGE_0_10     = 7
	MDK_MILLIAMPS_4_20   = 8
	MDK_OHM              = 10
	MDK_CO2              = 11
	MDK_BATTERY_VOLTAGE  = 12
	MDK_PUSH_FREQUENCY   = 13
	MDK_RAW              = 16
	MDK_UO               = 17
	MDK_UI               = 18
	MDK_DO               = 19
	MDK_DI               = 20
	MDK_FIRMWARE_VERSION = 61
	MDK_HARDWARE_VERSION = 62
	MDK_UINT_8           = 30
	MDK_INT_8            = 31
	MDK_UINT_16          = 32
	MDK_INT_16           = 33
	MDK_UINT_32          = 34
	MDK_INT_32           = 35
	MDK_UINT_64          = 36
	MDK_INT_64           = 37
	MDK_BOOL             = 38
	MDK_CHAR             = 39
	MDK_FLOAT            = 40
	MDK_DOUBLE           = 41
	MDK_STRING           = 42
)

const (
	FIXEDPOINT = 1
	DATAPOINT  = 2
)

const (
	SERIAL_DATA_MIN_SIZE         = 1
	SERIAL_DATA_DATA_OFFSET_BITS = 8
	SERIAL_DATA_DEFAULT_SETTINGS = 0
)

type SerialData struct {
	Buffer     []byte
	ReadBitPos int
}

func NewSerialData() *SerialData {
	buffer := make([]byte, SERIAL_DATA_MIN_SIZE)
	buffer[0] = SERIAL_DATA_DEFAULT_SETTINGS
	return &SerialData{
		Buffer:     buffer,
		ReadBitPos: SERIAL_DATA_DATA_OFFSET_BITS,
	}
}

func NewSerialDataWithBuffer(buffer []byte) *SerialData {
	return &SerialData{
		Buffer:     buffer,
		ReadBitPos: SERIAL_DATA_DATA_OFFSET_BITS,
	}
}

func canDecode(serialData *SerialData) bool {
	return serialData.ReadBitPos < (len(serialData.Buffer)*8 - HEADER_BIT_COUNT)
}

func setPositionalData(serialData *SerialData, set bool) {
	if set {
		// Set the first bit in the first byte to 1
		serialData.Buffer[0] |= 0x01
	} else {
		// Set the first bit in the first byte to 0
		serialData.Buffer[0] &^= 0x01
	}
}

func hasPositionalData(serialData *SerialData) bool {
	return serialData.Buffer[0]&1 == 1
}

func getHeader(serialData *SerialData, position *uint8) MetaDataKey {

	if hasPositionalData(serialData) {
		positionVector, shiftPos, bytesRequired := getVector(serialData, 8, serialData.ReadBitPos)
		*position = uint8(vectorToBits(positionVector, 8, shiftPos, bytesRequired))
	}

	typeVector, shiftPos, bytesRequired := getVector(serialData, HEADER_BIT_COUNT, serialData.ReadBitPos)
	return MetaDataKey(vectorToBits(typeVector, HEADER_BIT_COUNT, shiftPos, bytesRequired))
}

func vectorToBits(dataVector []byte, bitCount, shiftPos, bytesRequired int) uint64 {
	data64 := uint64(0)
	shiftLeft := (bytesRequired*8 - bitCount - shiftPos)
	for i := 0; i < len(dataVector)-1; i++ {
		data64 |= uint64(dataVector[i])
		if i != len(dataVector)-2 {
			data64 <<= 8
		}
	}
	// Shift forward and add last bits
	data64 <<= (8 - shiftLeft)
	data64 |= (uint64(dataVector[len(dataVector)-1]) >> shiftLeft)

	// Mask out leading bits
	if bitCount < 64 {
		data64 &= (1 << uint(bitCount)) - 1
	}

	return data64
}

func getMetaData(header MetaDataKey) MetaData {
	return serialMap[int(header)]
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

var serialMap = map[int]MetaData{
	MDK_TEMP:             {FIXEDPOINT, -45, 120, 2, 0},
	MDK_RH:               {FIXEDPOINT, 0, 100, 2, 0},
	MDK_LUX:              {FIXEDPOINT, 0, 65534, 0, 0},
	MDK_MOVEMENT:         {FIXEDPOINT, 0, 1, 0, 0},
	MDK_COUNTER:          {FIXEDPOINT, 0, 1048576, 0, 0},
	MDK_DIGITAL:          {FIXEDPOINT, 0, 1, 0, 0},
	MDK_VOLTAGE_0_10:     {FIXEDPOINT, 0, 10, 2, 0},
	MDK_MILLIAMPS_4_20:   {FIXEDPOINT, 4, 20, 2, 0},
	MDK_OHM:              {FIXEDPOINT, 0, 1048576, 0, 0},
	MDK_CO2:              {FIXEDPOINT, 0, 400, 0, 0},
	MDK_BATTERY_VOLTAGE:  {FIXEDPOINT, 0, 6, 1, 0},
	MDK_PUSH_FREQUENCY:   {FIXEDPOINT, 0, 2000, 0, 0},
	MDK_RAW:              {FIXEDPOINT, 0, 1, 3, 0},
	MDK_UO:               {FIXEDPOINT, 0, 1, 3, 0},
	MDK_UI:               {FIXEDPOINT, 0, 1, 3, 0},
	MDK_DO:               {FIXEDPOINT, 0, 1, 0, 0},
	MDK_DI:               {FIXEDPOINT, 0, 1, 0, 0},
	MDK_FIRMWARE_VERSION: {FIXEDPOINT, 0, 255, 0, 0},
	MDK_HARDWARE_VERSION: {FIXEDPOINT, 0, 255, 0, 0},
	MDK_UINT_8:           {DATAPOINT, 0, 0, 0, 1},
	MDK_INT_8:            {DATAPOINT, 0, 0, 0, 1},
	MDK_UINT_16:          {DATAPOINT, 0, 0, 0, 2},
	MDK_INT_16:           {DATAPOINT, 0, 0, 0, 2},
	MDK_UINT_32:          {DATAPOINT, 0, 0, 0, 4},
	MDK_INT_32:           {DATAPOINT, 0, 0, 0, 4},
	MDK_UINT_64:          {DATAPOINT, 0, 0, 0, 8},
	MDK_INT_64:           {DATAPOINT, 0, 0, 0, 8},
	MDK_BOOL:             {DATAPOINT, 0, 0, 0, 0},
	MDK_CHAR:             {DATAPOINT, 0, 0, 0, 1},
	MDK_FLOAT:            {DATAPOINT, 0, 0, 0, 4},
	MDK_DOUBLE:           {DATAPOINT, 0, 0, 0, 8}}

func DecodeRubix(data string, _ *LoRaDeviceDescription) (*CommonValues, interface{}) {
	/*
	 * Data Structure:
	 * -------------------------------------------------------------------------------------------------------------------------------------
	 * | 4 bytes address | 1 byte nonce | 1 byte length |            Payload           | 2 bytes RSSI | 2 bytes SNR |
	 * -----------------------------------------------------------------------------------------------------
	 * | data[0:4]       | data[4]      | data[5]       |       data[6:dataLen-4]      | data[dataLen-4:dataLen-2] | data[dataLen-2:dataLen]
	 * -------------------------------------------------------------------------------------------------------------------------------------
	 *
	 * - 4 bytes address:              data[0:4]
	 * - 1 byte nonce:                 data[4]
	 * - 1 byte length field:          data[5]
	 * - Payload:                      data[6:dataLen-4]
	 * - 2 bytes RSSI:                 data[dataLen-4:dataLen-2]
	 * - 2 bytes SNR:                  data[dataLen-2:dataLen]
	 */

	dataBytes, err := hex.DecodeString(data)
	if err != nil {
		fmt.Println("Error decoding hex string:", err)
		return nil, nil
	}
	payloadLength := len(dataBytes) - (4 + 1 + 1 + 4)
	payload := dataBytes[6 : 6+payloadLength]

	serialData := NewSerialDataWithBuffer(payload)

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
		fl2         float32
		fl3         float32
	)
	var float_count = 0
	for canDecode(serialData) {
		pos := uint8(0)
		header := getHeader(serialData, &pos)
		switch header {
		case MDK_TEMP:
			decodeData(serialData, header, &temperature)
		case MDK_RH:
			decodeData(serialData, header, &rh)
		case MDK_LUX:
			decodeData(serialData, header, &lux)
		case MDK_MOVEMENT:
			decodeData(serialData, header, &movement)
		case MDK_COUNTER:
			decodeData(serialData, header, &counter)
		case MDK_DIGITAL:
			decodeData(serialData, header, &digital)
		case MDK_VOLTAGE_0_10:
			decodeData(serialData, header, &voltage)
		case MDK_MILLIAMPS_4_20:
			decodeData(serialData, header, &amplitude)
		case MDK_OHM:
			decodeData(serialData, header, &ohm)
		case MDK_CO2:
			decodeData(serialData, header, &co2)
		case MDK_BATTERY_VOLTAGE:
			decodeData(serialData, header, &batVol)
		case MDK_PUSH_FREQUENCY:
			decodeData(serialData, header, &pushFreq)
		case MDK_RAW:
			decodeData(serialData, header, &raw)
		case MDK_UO:
			decodeData(serialData, header, &uo)
		case MDK_UI:
			decodeData(serialData, header, &ui)
		case MDK_DO:
			decodeData(serialData, header, &do)
		case MDK_DI:
			decodeData(serialData, header, &di)
		case MDK_FIRMWARE_VERSION:
			decodeData(serialData, header, &fwVer)
		case MDK_HARDWARE_VERSION:
			decodeData(serialData, header, &hwVer)
		case MDK_UINT_8:
			decodeData(serialData, header, &u8)
		case MDK_INT_8:
			decodeData(serialData, header, &i8)
		case MDK_UINT_16:
			decodeData(serialData, header, &u16)
		case MDK_INT_16:
			decodeData(serialData, header, &i16)
		case MDK_UINT_32:
			decodeData(serialData, header, &u32)
		case MDK_INT_32:
			decodeData(serialData, header, &i32)
		case MDK_UINT_64:
			decodeData(serialData, header, &u64)
		case MDK_INT_64:
			decodeData(serialData, header, &i64)
		case MDK_CHAR:
			decodeData(serialData, header, &char)
		case MDK_FLOAT:
			float_count++
			if float_count == 1 {
				decodeData(serialData, header, &fl1)
			} else if float_count == 2 {
				decodeData(serialData, header, &fl2)
			} else if float_count == 3 {
				decodeData(serialData, header, &fl3)
			}

		default:
			fmt.Printf("Unknown header: %d\n", header)
		}
	}

	v :=
		TRubix{
			Temp:      temperature,
			Rh:        rh,
			Lux:       lux,
			Movement:  movement,
			Count:     counter,
			Digital:   digital,
			Voltage:   voltage,
			Amplitude: amplitude,
			Ohm:       ohm,
			Co2:       co2,
			BatVol:    batVol,
			PushFreq:  pushFreq,
			Raw:       raw,
			Uo:        uo,
			Ui:        ui,
			Do:        do,
			Di:        di,
			FwVer:     fwVer,
			HwVer:     hwVer,
			Uint8:     u8,
			Int8:      i8,
			Uint16:    u16,
			Int16:     i16,
			Uint32:    u32,
			Int32:     i32,
			Uint64:    u64,
			Int64:     i64,
			Char:      char,
			Float1:    fl1,
			Float2:    fl2,
			Float3:    fl3,
		}
	return &v.CommonValues, v
}

func GetPointsStructRubix() interface{} {
	return TRubix{}
}

func CheckPayloadLengthRubix(data string) bool {
	// 4 bytes address | 1 byte nonce | 1 byte length | 2 byte rssi | 2 byte snr
	payloadLength := len(data) - 20
	payloadLength /= 2
	dataLength, _ := strconv.ParseInt(data[10:12], 16, 0)

	return payloadLength == int(dataLength)
}
