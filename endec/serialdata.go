package endec

import (
	"math/rand"
)

type SerialData struct {
	Buffer      []byte
	ReadBitPos  int
	WriteBitPos int
}

const (
	MinSize         = 1
	DataOffsetBits  = 8
	DefaultSettings = 0
)

func NewSerialData() *SerialData {
	buffer := make([]byte, MinSize)
	buffer[0] = DefaultSettings
	return &SerialData{
		Buffer:      buffer,
		ReadBitPos:  DataOffsetBits,
		WriteBitPos: DataOffsetBits,
	}
}

func NewSerialDataWithBuffer(buffer []byte) *SerialData {
	return &SerialData{
		Buffer:      buffer,
		ReadBitPos:  DataOffsetBits,
		WriteBitPos: DataOffsetBits,
	}
}

func BIT_SET(byteValue byte, bit bool, position uint8) byte {
	if bit {
		return byteValue | (1 << position)
	}
	return byteValue &^ (1 << position)
}

func SetPositionalData(serialData *SerialData, set bool) {
	serialData.Buffer[0] = BIT_SET(serialData.Buffer[0], set, 0)
}

func HasPositionalData(serialData *SerialData) bool {
	return serialData.Buffer[0]&1 == 1
}

func SetRequestData(serialData *SerialData, set bool) {
	serialData.Buffer[0] = BIT_SET(serialData.Buffer[0], set, 1)
}

func HasRequestData(serialData *SerialData) bool {
	return serialData.Buffer[0]&2 != 0
}

func SetResponseData(serialData *SerialData, set bool) {
	serialData.Buffer[0] = BIT_SET(serialData.Buffer[0], set, 2)
}

func HasResponseData(serialData *SerialData) bool {
	return serialData.Buffer[0]&4 != 0
}

func UpdateBitPositionsForHeaderByte(serialData *SerialData) {
	if HasRequestData(serialData) || HasResponseData(serialData) {
		serialData.ReadBitPos += 8 // 8 bits for message ID
	}
}

func SetMessageId(serialData *SerialData, id uint8) {
	if HasRequestData(serialData) || HasResponseData(serialData) {
		serialData.Buffer = append(serialData.Buffer, id)
	}
}

func GetMessageId(serialData *SerialData) uint8 {
	if HasRequestData(serialData) || HasResponseData(serialData) {
		return serialData.Buffer[1]
	}
	return 0
}

func GenerateRandomId() (uint8, error) {
	// Create a new Rand instance with a seed
	var b [1]byte
	_, err := rand.Read(b[:])
	if err != nil {
		return 0, err
	}
	return b[0], nil
}
