package decoder

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
