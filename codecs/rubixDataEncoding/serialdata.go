package rubixDataEncoding

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

func setPositionalData(serialData *SerialData, set bool) {
	serialData.Buffer[0] = BIT_SET(serialData.Buffer[0], set, 0)
}

func hasPositionalData(serialData *SerialData) bool {
	return serialData.Buffer[0]&1 == 1
}
