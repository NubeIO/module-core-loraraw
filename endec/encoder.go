package endec

import (
	"errors"
	"github.com/NubeIO/lib-utils-go/nstring"
	"github.com/NubeIO/module-core-loraraw/aesutils"
	"github.com/NubeIO/module-core-loraraw/utils"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
	"math"
	"strconv"
	"unsafe"

	log "github.com/sirupsen/logrus"
)

func reshapeVector(serialData *SerialData, bitCount int, dataVector []byte) []byte {
	if (serialData.WriteBitPos + bitCount) > len(dataVector)*8 {
		dataVector = append(dataVector, 0)
	}

	var TB uint64 = 0
	for i := range dataVector {
		TB = ((TB | uint64(dataVector[i])) << 8) >> uint64(serialData.WriteBitPos)
		dataVector[i] = byte(TB >> 8)
		TB = TB << uint64(serialData.WriteBitPos)
	}

	return dataVector
}

func addVectorToBuffer(serialData *SerialData, dataVector []byte, bitCount int) bool {
	dataVector = reshapeVector(serialData, bitCount, dataVector)

	// Handle first byte
	if serialData.WriteBitPos == 0 {
		serialData.Buffer = append(serialData.Buffer, dataVector[0])
	} else {
		combinedByte := serialData.Buffer[len(serialData.Buffer)-1] | dataVector[0]
		serialData.Buffer[len(serialData.Buffer)-1] = combinedByte
	}

	// Add rest of bytes to array
	serialData.Buffer = append(serialData.Buffer, dataVector[1:]...)

	// Update bit write position
	offsetBitCount := bitCount % 8
	serialData.WriteBitPos = (serialData.WriteBitPos + offsetBitCount) % 8

	return true
}

func bitsToVector(data64 uint64, bitCount int, dataVector *[]byte) {
	// Calculate number of bytes required
	numberOfBytes := int(math.Ceil(float64(bitCount) / 8.0))

	// Shift all data64 to left most bit
	leftShift := (numberOfBytes * 8) - bitCount
	data64 <<= uint(leftShift)

	// Build byte vector from data64
	for i := numberOfBytes - 1; i >= 0; i-- {
		byteVector := byte((data64 >> uint(8*i)) & 0xFF)
		*dataVector = append(*dataVector, byteVector)
	}
}

func clamp[T float64 | float32](value, min, max T) T {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func fixedPointToBits[T float64 | float32 | uint16 | uint32 | uint64](value T, metaData *MetaData, data64 *uint64, bitCount *int) bool {
	low := float64(metaData.lowValue)
	high := float64(metaData.highValue)
	decimal := metaData.decimalPoint

	// Clamp data to ensure it's within range
	value = T(clamp(float64(value), low, high))

	// Get number of bits required
	*bitCount = getBitCount(metaData.lowValue, metaData.highValue, decimal)

	// Convert data to uint64
	*data64 = uint64(math.Round(float64(value-T(low)) * math.Pow10(decimal)))

	return true
}

func dataTypeToBits[T any](data T, metaData *MetaData, data64 *uint64, bitCount *int) bool {
	dataSize := int(unsafe.Sizeof(data))

	if metaData.byteCount != dataSize {
		return false
	}

	// Use unsafe.Pointer to directly copy memory
	*data64 = 0
	dataPtr := unsafe.Pointer(&data)
	data64Ptr := unsafe.Pointer(data64)
	for i := 0; i < dataSize && i < 8; i++ {
		*(*byte)(unsafe.Add(data64Ptr, i)) = *(*byte)(unsafe.Add(dataPtr, i))
	}

	*bitCount = metaData.byteCount * 8

	return true
}

func encodeData[T any](serialData *SerialData, data T, header MetaDataKey, position uint8) bool {
	metaData := getMetaData(header)
	headerVector := make([]byte, 0)
	dataVector := make([]byte, 0)
	var bitCount int
	headerBitCount := HEADER_BIT_COUNT
	// Build header vector
	if HasPositionalData(serialData) {
		headerVector = append(headerVector, position)
		headerBitCount += 8
	}
	bitsToVector(uint64(header), HEADER_BIT_COUNT, &headerVector)
	switch metaData.dataType {
	case FIXEDPOINT:
		var dataBits uint64
		// Convert data to uint64
		switch v := any(data).(type) {
		case float64:
			if !fixedPointToBits(v, &metaData, &dataBits, &bitCount) {
				log.Errorf("encodeData: fixedPointToBits failed for float64")
				return false
			}
		case float32:
			if !fixedPointToBits(v, &metaData, &dataBits, &bitCount) {
				log.Errorf("encodeData: fixedPointToBits failed for float32")
				return false
			}
		default:
			log.Errorf("%v", v)
			log.Errorf("encodeData: Unsupported type for FIXEDPOINT: %T", data)
			return false
		}
		// Add header to buffer
		addVectorToBuffer(serialData, headerVector, headerBitCount)
		// Convert uint64 to vector
		bitsToVector(dataBits, bitCount, &dataVector)
	case DATAPOINT:
		var dataBits uint64
		// Convert data to uint64, get bitCount
		if !dataTypeToBits(data, &metaData, &dataBits, &bitCount) {
			log.Errorf("dataTypeToBits")
			return false
		}
		// Add header to buffer
		addVectorToBuffer(serialData, headerVector, headerBitCount)
		// Convert uint64 to vector
		bitsToVector(dataBits, bitCount, &dataVector)

	default:
		return false
	}

	// Add dataVector to serial buffer
	addVectorToBuffer(serialData, dataVector, bitCount)

	return true
}

func EncodeAndEncrypt(point *model.Point, serialData *SerialData, key []byte) ([]byte, error) {
	writeValue := point.WriteValue
	if writeValue == nil {
		return nil, errors.New("encoding failed for nil point")
	}

	pointDataType, err := strconv.Atoi(point.DataType)
	if err != nil {
		return nil, err
	}

	addressID, err := utils.SafeDereferenceUint8(point.AddressID)
	if err != nil {
		return nil, err
	}

	if MetaDataKey(pointDataType) == MDK_UINT_8 {
		encodeData(serialData, uint8(*writeValue), MetaDataKey(pointDataType), addressID)
	} else if MetaDataKey(pointDataType) == MDK_UINT_16 {
		encodeData(serialData, uint16(*writeValue), MetaDataKey(pointDataType), addressID)
	} else if MetaDataKey(pointDataType) == MDK_UINT_32 {
		encodeData(serialData, uint32(*writeValue), MetaDataKey(pointDataType), addressID)
	} else if MetaDataKey(pointDataType) == MDK_UINT_64 {
		encodeData(serialData, uint64(*writeValue), MetaDataKey(pointDataType), addressID)
	} else if MetaDataKey(pointDataType) == MDK_INT_8 {
		encodeData(serialData, int8(*writeValue), MetaDataKey(pointDataType), addressID)
	} else if MetaDataKey(pointDataType) == MDK_INT_16 {
		encodeData(serialData, int16(*writeValue), MetaDataKey(pointDataType), addressID)
	} else if MetaDataKey(pointDataType) == MDK_INT_32 {
		encodeData(serialData, int32(*writeValue), MetaDataKey(pointDataType), addressID)
	} else if MetaDataKey(pointDataType) == MDK_INT_64 {
		encodeData(serialData, int64(*writeValue), MetaDataKey(pointDataType), addressID)
	} else {
		encodeData(serialData, *writeValue, MetaDataKey(pointDataType), addressID)
	}

	encryptedData, err := aesutils.Encrypt(
		nstring.DerefString(point.AddressUUID),
		serialData.Buffer,
		key,
		0,
	)
	if err != nil {
		return nil, err
	}

	return encryptedData, nil
}
