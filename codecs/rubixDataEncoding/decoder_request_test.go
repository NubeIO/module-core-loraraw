package rubixDataEncoding

import "testing"

// buildRequestPayload constructs a §2.2 request body the way the firmware
// does: settings byte with the request flag, the RDE message ID, then one
// POINT_ID byte per requested point.
func buildRequestPayload(msgID uint8, positions []uint8) []byte {
	sd := NewSerialData()
	SetRequestData(sd, true)
	SetMessageId(sd, msgID)
	sd.Buffer = append(sd.Buffer, positions...)
	return sd.Buffer
}

func TestDecodeRequestPayloadSinglePoint(t *testing.T) {
	// UVP-1 => type UVP in the high 3 bits, index 0 in the low 5.
	uvp1 := uint8(PositionDataType_UVP)<<5 | 0
	got, err := DecodeRequestPayload(buildRequestPayload(0x42, []uint8{uvp1}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0] != "UVP-1" {
		t.Fatalf("got %v, want [UVP-1]", got)
	}
}

func TestDecodeRequestPayloadMultiplePoints(t *testing.T) {
	uvp1 := uint8(PositionDataType_UVP)<<5 | 0
	// UVP-40 => pointIdx 39 is >=32, so per getPosition/generateFieldName it is
	// encoded as PositionDataType_UVP2 with ID 39-32=7 (id+32 => 40). The ID
	// field is only 5 bits (0-31), so PositionDataType_UVP<<5|39 would collide
	// with the type bits and decode to the wrong point.
	uvp40 := uint8(PositionDataType_UVP2)<<5 | 7
	got, err := DecodeRequestPayload(buildRequestPayload(0x07, []uint8{uvp1, uvp40}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 || got[0] != "UVP-1" || got[1] != "UVP-40" {
		t.Fatalf("got %v, want [UVP-1 UVP-40]", got)
	}
}

// A payload without the request flag is a data packet, not a request.
func TestDecodeRequestPayloadRejectsNonRequest(t *testing.T) {
	if _, err := DecodeRequestPayload([]byte{0x00, 0x20}); err == nil {
		t.Fatal("expected an error when the request flag is clear")
	}
}

func TestDecodeRequestPayloadRejectsTruncated(t *testing.T) {
	// Request flag set but the message ID byte never arrived.
	if _, err := DecodeRequestPayload([]byte{0x02}); err == nil {
		t.Fatal("expected an error on a truncated request")
	}
}

func TestDecodeRequestPayloadRejectsEmpty(t *testing.T) {
	if _, err := DecodeRequestPayload(nil); err == nil {
		t.Fatal("expected an error on an empty payload")
	}
}

// No POINT_IDs after the header is well-formed but empty; it must not panic.
func TestDecodeRequestPayloadEmptyPointList(t *testing.T) {
	got, err := DecodeRequestPayload(buildRequestPayload(0x01, nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("got %v, want empty", got)
	}
}
