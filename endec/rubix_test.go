package endec

import (
	"reflect"
	"testing"
)

func testHeader(t *testing.T, serialData *SerialData, expectedHeader MetaDataKey) {
	var pos uint8
	header := getHeader(serialData, &pos)
	if header != expectedHeader {
		t.Fatalf("Expected header %v, but got %v", expectedHeader, header)
	}
}

func testFloat(t *testing.T, serialData *SerialData, expected_header MetaDataKey, expected float32) {
	var value float32
	err := decodeData(serialData, expected_header, &value)
	if err != nil {
		t.Fatalf("Failed to decode data: %v", err)
	}
	if value != expected {
		t.Fatalf("Expected float value %v, but got %v", expected, value)
	}
}

func testDecodeFloat(t *testing.T, serialData *SerialData, expected float32, h MetaDataKey) {
	if !canDecode(serialData) {
		t.Fatalf("Expected canDecode to return true, but it returned false")
	}

	testHeader(t, serialData, h)
	testFloat(t, serialData, h, expected)
}

func testUint(t *testing.T, serialData *SerialData, expectedHeader MetaDataKey, expected interface{}) {
	value := reflect.New(reflect.TypeOf(expected)).Interface()
	err := decodeData(serialData, expectedHeader, value)
	if err != nil {
		t.Fatalf("Failed to decode data: %v", err)
	}

	if !reflect.DeepEqual(reflect.ValueOf(value).Elem().Interface(), expected) {
		t.Fatalf("Expected value %v, but got %v", expected, reflect.ValueOf(value).Elem().Interface())
	}
}

func testDecodeUint(t *testing.T, serialData *SerialData, expected interface{}, h MetaDataKey) {
	if !canDecode(serialData) {
		t.Fatalf("Expected canDecode to return true, but it returned false")
	}

	testHeader(t, serialData, h)
	testUint(t, serialData, h, expected)
}

func testInt(t *testing.T, serialData *SerialData, expectedHeader MetaDataKey, expected interface{}) {
	value := reflect.New(reflect.TypeOf(expected)).Interface()
	err := decodeData(serialData, expectedHeader, value)
	if err != nil {
		t.Fatalf("Failed to decode data: %v", err)
	}

	if !reflect.DeepEqual(reflect.ValueOf(value).Elem().Interface(), expected) {
		t.Fatalf("Expected value %v, but got %v", expected, reflect.ValueOf(value).Elem().Interface())
	}
}

func testDecodeInt(t *testing.T, serialData *SerialData, expected interface{}, h MetaDataKey) {
	if !canDecode(serialData) {
		t.Fatalf("Expected canDecode to return true, but it returned false")
	}

	testHeader(t, serialData, h)
	testInt(t, serialData, h, expected)
}

func testChar(t *testing.T, serialData *SerialData, expectedHeader MetaDataKey, expected byte) {
	var value byte
	err := decodeData(serialData, expectedHeader, &value)
	if err != nil {
		t.Fatalf("Failed to decode data: %v", err)
	}
	if value != expected {
		t.Fatalf("Expected char value %v, but got %v", expected, value)
	}
}

func testDecodeChar(t *testing.T, serialData *SerialData, expected byte, h MetaDataKey) {
	if !canDecode(serialData) {
		t.Fatalf("Expected canDecode to return true, but it returned false")
	}

	testHeader(t, serialData, h)
	testChar(t, serialData, h, expected)
}

func expectTrue(t *testing.T, serialData *SerialData) {
	if !canDecode(serialData) {
		t.Fatalf("Expected canDecode to return true, but it returned false")
	}
}

func expectFalse(t *testing.T, serialData *SerialData) {
	if canDecode(serialData) {
		t.Fatalf("Expected canDecode to return false, but it returned true")
	}
}

func requireFalse(t *testing.T, condition bool, msg string) {
	if condition {
		t.Fatalf("REQUIRE_FALSE failed: %s", msg)
	}
}

func TestDecodefromStm32Encode(t *testing.T) {
	pl := []byte{0, 5, 92, 240, 74, 217, 134, 205, 44, 36, 83, 13, 63,
		26, 62, 240, 68, 192, 41, 178, 7, 11, 166, 152, 233, 160,
		61, 13, 227, 209, 111, 139, 30, 1, 123, 253, 229, 7, 236,
		31, 0, 120, 5, 224, 39, 192, 222, 15, 124, 125, 227, 247,
		223, 223, 128, 121, 65, 251, 8, 1, 144, 97, 249, 191, 136,
		0, 19, 136, 6, 63, 255, 177, 223, 249, 0, 0, 0, 3,
		185, 172, 160, 0, 101, 255, 255, 255, 241, 25, 77, 127, 255,
		148, 0, 0, 0, 0, 0, 0, 0, 2, 95, 255, 255, 255,
		255, 255, 255, 255, 249, 216, 104, 63, 128, 0, 0, 161, 12,
		72, 125, 246, 132, 53, 227, 141, 90, 16, 233, 170, 168, 0}

	serialData := SerialData{
		Buffer:     pl,
		ReadBitPos: 8,
	}

	Require(t, canDecode(&serialData))
	testDecodeFloat(t, &serialData, 66.66, MDK_TEMP)
	testDecodeFloat(t, &serialData, 55.55, MDK_RH)
	testDecodeFloat(t, &serialData, 26262, MDK_LUX)
	testDecodeFloat(t, &serialData, 1, MDK_MOVEMENT)
	testDecodeFloat(t, &serialData, 199999, MDK_COUNTER)
	testDecodeFloat(t, &serialData, 1, MDK_DIGITAL)
	testDecodeFloat(t, &serialData, 8.88, MDK_VOLTAGE_0_10)
	testDecodeFloat(t, &serialData, 16.16, MDK_MILLIAMPS_4_20)
	testDecodeFloat(t, &serialData, 444444, MDK_OHM)
	testDecodeFloat(t, &serialData, 333, MDK_CO2)
	testDecodeFloat(t, &serialData, 2.9, MDK_BATTERY_VOLTAGE)
	testDecodeFloat(t, &serialData, 15, MDK_PUSH_FREQUENCY)
	testDecodeFloat(t, &serialData, 0.888, MDK_RAW)
	testDecodeFloat(t, &serialData, 22, MDK_FIRMWARE_VERSION)
	testDecodeFloat(t, &serialData, 44, MDK_HARDWARE_VERSION)
	testDecodeUint(t, &serialData, (uint8)(1), MDK_UINT_8)
	testDecodeUint(t, &serialData, (uint8)(0xFF), MDK_UINT_8)
	testDecodeUint(t, &serialData, (uint8)(80), MDK_UINT_8)
	testDecodeInt(t, &serialData, int8(-80), MDK_INT_8)
	testDecodeInt(t, &serialData, int8(0), MDK_INT_8)
	testDecodeUint(t, &serialData, (uint8)(1), MDK_UINT_8)
	testDecodeUint(t, &serialData, (uint8)(2), MDK_UINT_8)
	testDecodeInt(t, &serialData, int8(3), MDK_INT_8)
	testDecodeUint(t, &serialData, (uint8)(15), MDK_UINT_8)
	testDecodeInt(t, &serialData, int8(31), MDK_INT_8)
	testDecodeUint(t, &serialData, (uint8)(63), MDK_UINT_8)
	testDecodeInt(t, &serialData, int8(127), MDK_INT_8)
	testDecodeInt(t, &serialData, int8(-128), MDK_INT_8)
	testDecodeUint(t, &serialData, (uint8)(80), MDK_UINT_8)
	testDecodeInt(t, &serialData, int8(-80), MDK_INT_8)
	testDecodeUint(t, &serialData, (uint16)(1601), MDK_UINT_16)
	testDecodeInt(t, &serialData, int16(-1601), MDK_INT_16)
	testDecodeUint(t, &serialData, uint32(320001), MDK_UINT_32)
	testDecodeInt(t, &serialData, int32(-320001), MDK_INT_32)
	testDecodeUint(t, &serialData, uint64(64000000001), MDK_UINT_64)
	testDecodeInt(t, &serialData, int64(-64000000001), MDK_INT_64)
	testDecodeInt(t, &serialData, int64(0), MDK_INT_64)
	testDecodeInt(t, &serialData, int64(-1), MDK_INT_64)
	testDecodeChar(t, &serialData, byte('a'), MDK_CHAR)
	testDecodeFloat(t, &serialData, (1.0), MDK_FLOAT)
	testDecodeFloat(t, &serialData, 146.123, MDK_FLOAT)
	testDecodeFloat(t, &serialData, 222.222, MDK_FLOAT)
	testDecodeFloat(t, &serialData, 333.333, MDK_FLOAT)
	requireFalse(t, canDecode(&serialData), "Should not decode")
}

func Require(t *testing.T, condition bool) {
	if !condition {
		t.Fatalf("REQUIRE failed:")
	}
}

func TestSerialDataSettings(t *testing.T) {
	sd := NewSerialData()
	Require(t, len(sd.Buffer) == 1)
	Require(t, hasPositionalData(sd) == false)
	setPositionalData(sd, true)
	Require(t, sd.Buffer[0] == 1)
	Require(t, hasPositionalData(sd) == true)
	setPositionalData(sd, false)
	Require(t, sd.Buffer[0] == 0)
	Require(t, hasPositionalData(sd) == false)
}

func TestSerialDataRaw(t *testing.T) {
	pl := []byte{0, 64, 100, 64, 200, 65, 44, 65, 144, 65, 244, 64, 100,
		64, 200, 65, 44, 65, 144, 65, 244, 66, 88, 26, 52}
	serialData := NewSerialDataWithBuffer(pl)

	Require(t, canDecode(serialData))

	testDecodeFloat(t, serialData, 0.1, MDK_RAW)
	testDecodeFloat(t, serialData, 0.2, MDK_RAW)
	testDecodeFloat(t, serialData, 0.3, MDK_RAW)
	testDecodeFloat(t, serialData, 0.4, MDK_RAW)
	testDecodeFloat(t, serialData, 0.5, MDK_RAW)
	testDecodeFloat(t, serialData, 0.1, MDK_RAW)
	testDecodeFloat(t, serialData, 0.2, MDK_RAW)
	testDecodeFloat(t, serialData, 0.3, MDK_RAW)
	testDecodeFloat(t, serialData, 0.4, MDK_RAW)
	testDecodeFloat(t, serialData, 0.5, MDK_RAW)
	testDecodeFloat(t, serialData, 0.6, MDK_RAW)
	testDecodeFloat(t, serialData, 1, MDK_DIGITAL)
	testDecodeFloat(t, serialData, 1, MDK_DIGITAL)

	requireFalse(t, canDecode(serialData), "Should not decode")
}

func TestDecodeDataType(t *testing.T) {
	pl := []byte{0, 121, 65, 251, 8, 1, 144, 97, 249, 191, 136, 0, 19, 136, 6,
		63, 255, 177, 223, 249, 216, 104, 63, 143, 190, 119, 162, 254,
		62, 249, 222, 132, 71, 156, 5, 32}
	serialData := NewSerialDataWithBuffer(pl)

	Require(t, canDecode(serialData))

	testDecodeUint(t, serialData, uint8(80), MDK_UINT_8)
	testDecodeInt(t, serialData, int8(-80), MDK_INT_8)
	testDecodeUint(t, serialData, uint16(1601), MDK_UINT_16)
	testDecodeInt(t, serialData, int16(-1601), MDK_INT_16)
	testDecodeUint(t, serialData, uint32(320001), MDK_UINT_32)
	testDecodeInt(t, serialData, int32(-320001), MDK_INT_32)
	testDecodeChar(t, serialData, byte('a'), MDK_CHAR)
	testDecodeFloat(t, serialData, 1.123, MDK_FLOAT)
	testDecodeFloat(t, serialData, -1.123, MDK_FLOAT)
	testDecodeFloat(t, serialData, 999.005, MDK_FLOAT)
	requireFalse(t, canDecode(serialData), "Should not decode")
}

func TestPositional(t *testing.T) {
	pl := []byte{1, 1, 64, 100, 3, 65, 44, 5, 65, 244, 7, 66, 188}
	serialData := NewSerialDataWithBuffer(pl)

	Require(t, canDecode(serialData))

	testDecodeFloat(t, serialData, 0.1, MDK_RAW)
	testDecodeFloat(t, serialData, 0.3, MDK_RAW)
	testDecodeFloat(t, serialData, 0.5, MDK_RAW)
	testDecodeFloat(t, serialData, 0.7, MDK_RAW)
	requireFalse(t, canDecode(serialData), "Should not decode")
}

func TestSerialDataFull(t *testing.T) {
	pl := []byte{0, 5, 92, 240, 74, 217, 134, 205, 44, 36, 83, 13, 63, 26,
		62, 240, 68, 192, 41, 178, 7, 11, 166, 152, 233, 160, 61, 13,
		225, 17, 145, 35, 33, 50, 159, 69, 190, 44}
	serialData := NewSerialDataWithBuffer(pl)

	Require(t, canDecode(serialData))

	testDecodeFloat(t, serialData, 66.66, MDK_TEMP)
	testDecodeFloat(t, serialData, 55.55, MDK_RH)
	testDecodeFloat(t, serialData, 26262, MDK_LUX)
	testDecodeFloat(t, serialData, 1, MDK_MOVEMENT)
	testDecodeFloat(t, serialData, 199999, MDK_COUNTER)
	testDecodeFloat(t, serialData, 1, MDK_DIGITAL)
	testDecodeFloat(t, serialData, 8.88, MDK_VOLTAGE_0_10)
	testDecodeFloat(t, serialData, 16.16, MDK_MILLIAMPS_4_20)
	testDecodeFloat(t, serialData, 444444, MDK_OHM)
	testDecodeFloat(t, serialData, 333, MDK_CO2)
	testDecodeFloat(t, serialData, 2.9, MDK_BATTERY_VOLTAGE)
	testDecodeFloat(t, serialData, 15, MDK_PUSH_FREQUENCY)
	testDecodeFloat(t, serialData, 0.888, MDK_RAW)

	testDecodeFloat(t, serialData, 0.1, MDK_UO)
	testDecodeFloat(t, serialData, 0.2, MDK_UI)
	testDecodeFloat(t, serialData, 0, MDK_DO)
	testDecodeFloat(t, serialData, 1, MDK_DI)

	testDecodeFloat(t, serialData, 22, MDK_FIRMWARE_VERSION)
	testDecodeFloat(t, serialData, 44, MDK_HARDWARE_VERSION)

	requireFalse(t, canDecode(serialData), "Should not decode")
}
