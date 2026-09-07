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

const (
	ErrorCodeNone         = 0
	ErrorCodeGeneral      = 1
	ErrorCodeNotAllowed   = 2
	ErrorCodeWriteFailed  = 3
	ErrorCodeInvalidPoint = 4
	ErrorCodeInvalidType  = 5
	ErrorCodeInvalidValue = 6
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

// SETTINGS_BYTE bit layout (LORA RAW PROTOCOL §3.3, RDE v1.1.x). Mirrors
// Core/Inc/serialData.hpp in the device firmware; keep the two in step.
//
//	bit 0 - positional flag: POINT_IDs are present in the data packets
//	bit 1 - request flag:    payload is POINT_IDs only, no DATA_TYPE_ID/values
//	bit 2 - response flag:   payload answers a request
//
// When bit 1 or bit 2 is set, Buffer[1] holds the RDE message ID and the
// first data packet starts one byte later.

func SetRequestData(serialData *SerialData, set bool) {
	serialData.Buffer[0] = BIT_SET(serialData.Buffer[0], set, 1)
	SetMessageId(serialData, 0)
}

func HasRequestData(serialData *SerialData) bool {
	return serialData.Buffer[0]&2 == 2
}

func SetResponseData(serialData *SerialData, set bool) {
	serialData.Buffer[0] = BIT_SET(serialData.Buffer[0], set, 2)
	SetMessageId(serialData, 0)
}

func HasResponseData(serialData *SerialData) bool {
	return serialData.Buffer[0]&4 == 4
}

func HasPositionalData(serialData *SerialData) bool {
	return hasPositionalData(serialData)
}

// SetMessageId writes the RDE message ID, reserving Buffer[1] if the buffer
// has not grown that far yet. It is a no-op unless a request or response flag
// is set, matching the firmware's setMessageId precondition.
func SetMessageId(serialData *SerialData, id uint8) {
	if !HasRequestData(serialData) && !HasResponseData(serialData) {
		return
	}
	if len(serialData.Buffer) > 1 {
		serialData.Buffer[1] = id
		return
	}
	serialData.Buffer = append(serialData.Buffer, id)
}

func GetMessageId(serialData *SerialData) uint8 {
	if len(serialData.Buffer) < 2 {
		return 0
	}
	return serialData.Buffer[1]
}

// UpdateBitPositionsForHeaderByte advances the read position past the RDE
// message ID byte. Must be called before canDecode/decodeData on any buffer
// that may carry a request or response flag. No-op when both flags are clear,
// so plain uplink decoding is unaffected.
func UpdateBitPositionsForHeaderByte(serialData *SerialData) {
	if HasRequestData(serialData) || HasResponseData(serialData) {
		serialData.ReadBitPos += 8
	}
}
