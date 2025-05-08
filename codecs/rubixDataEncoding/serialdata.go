package rubixDataEncoding

type SerialData struct {
	Buffer      []byte
	ReadBitPos  int
	WriteBitPos int
}

type PositionDataType int

const (
	PositionDataType_GENERAL PositionDataType = iota
	PositionDataType_UO      PositionDataType = iota
	PositionDataType_DO      PositionDataType = iota
	PositionDataType_UI      PositionDataType = iota
	PositionDataType_DI      PositionDataType = iota
	PositionDataType_UVP     PositionDataType = iota
	PositionDataType_UVP2    PositionDataType = iota
	PositionDataType_DVP     PositionDataType = iota
)

type PositionData struct {
	ID   int
	Type PositionDataType
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
