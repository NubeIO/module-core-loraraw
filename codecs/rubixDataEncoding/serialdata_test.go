package rubixDataEncoding

import "testing"

func TestRequestFlagRoundTrip(t *testing.T) {
	sd := NewSerialData()
	if HasRequestData(sd) {
		t.Fatal("request flag should start clear")
	}
	SetRequestData(sd, true)
	if !HasRequestData(sd) {
		t.Fatal("request flag should be set")
	}
	if HasResponseData(sd) {
		t.Fatal("setting request must not set response")
	}
	if HasPositionalData(sd) {
		t.Fatal("setting request must not set positional")
	}
}

// SetRequestData must reserve Buffer[1] for the message ID, mirroring the
// firmware's setRequestData -> setMessageId(0) behaviour.
func TestSetRequestDataReservesMessageIdByte(t *testing.T) {
	sd := NewSerialData()
	SetRequestData(sd, true)
	if len(sd.Buffer) < 2 {
		t.Fatalf("expected a message ID byte at Buffer[1], buffer len = %d", len(sd.Buffer))
	}
	SetMessageId(sd, 0x5A)
	if got := GetMessageId(sd); got != 0x5A {
		t.Fatalf("message ID = %#x, want 0x5A", got)
	}
}

// With both flags clear the read position must be untouched, so existing
// uplink decoding is byte-identical to before this change.
func TestUpdateBitPositionsIsNoopWhenFlagsClear(t *testing.T) {
	sd := NewSerialDataWithBuffer([]byte{0x00, 0x11, 0x22})
	before := sd.ReadBitPos
	UpdateBitPositionsForHeaderByte(sd)
	if sd.ReadBitPos != before {
		t.Fatalf("ReadBitPos moved from %d to %d with flags clear", before, sd.ReadBitPos)
	}
}

func TestUpdateBitPositionsSkipsMessageIdByte(t *testing.T) {
	for _, tc := range []struct {
		name string
		set  func(*SerialData)
	}{
		{"request", func(sd *SerialData) { SetRequestData(sd, true) }},
		{"response", func(sd *SerialData) { SetResponseData(sd, true) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sd := NewSerialData()
			tc.set(sd)
			before := sd.ReadBitPos
			UpdateBitPositionsForHeaderByte(sd)
			if sd.ReadBitPos != before+8 {
				t.Fatalf("ReadBitPos = %d, want %d", sd.ReadBitPos, before+8)
			}
		})
	}
}
