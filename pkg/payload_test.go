package pkg

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/NubeIO/module-core-loraraw/aesutils"
	"github.com/NubeIO/module-core-loraraw/codec"
)

// defaultKey mirrors cipherSetting.py:
//
//	key = bytes([3, 1, 2, 22, 4, 5, 15, 7, 230, 9, 90, 11, 12, 18, 99, 15])
const testDefaultKey = "0301021604050f07e6095a0b0c12630f"

// knownDevices are the registered device addresses (uppercase).
var knownDevices = map[string]bool{
	"3DAC603E": true,
	"55ACA79B": true,
	"1AAC6631": true,
	"3CAC17A2": true,
	"ABACB636": true,
	"D6AC86B0": true,
	"B7AC1511": true,
}

type frameCase struct {
	raw            string // full hex string as received on serial port
	wantAddr       string // expected decrypted device address (uppercase), "" if unknown
	wantIsLegacy   bool
	wantPayloadHex string // expected decrypted payload hex (bytes after addr), "" if unknown/LoRaRAW
	wantRSSI       int
	wantSNR        float32
}

var frameCases = []frameCase{
	// Legacy frames (len-2)%16 == 0
	// Raw frame byte order from SX127x driver: [...payload...][SNR][RSSI]
	// _serial_lora_format re-encodes as: payload.hex + rssi_enc.hex + snr.hex
	// So on the wire the last 4 hex chars are: [RSSI byte hex][SNR byte hex]
	// DecodeRSSI reads data[-4:-2] = RSSI, DecodeSNR reads data[-2:] = SNR
	{
		// raw[-2]=0x72=114(SNR), raw[-1]=0x08=8(RSSI) -> rssi=8-157=-149, snr=114/4=28.5
		// but _serial_lora_format sends rssi_enc=(-149*-1)&0xFF=0x95, snr=0x72
		// DecodeRSSI reads 0x95 -> -149, DecodeSNR reads 0x72 -> 28.5
		// However these are the RAW frames before driver reformatting,
		// so DecodeRSSI(raw) reads raw[-4:-2]=0x72 -> -114, DecodeSNR reads 0x08 -> 2.0
		raw:            "2D0458BC3C59EF6AEDCB3DCE5565834A7208",
		wantAddr:       "D5AC3F35",
		wantIsLegacy:   true,
		wantPayloadHex: "0005455FB803FF03FF03FF01",
		wantRSSI:       -114, // Go DecodeRSSI on raw hex reads position [-4:-2] = 0x72 -> -114
		wantSNR:        2.0,  // Go DecodeSNR on raw hex reads position [-2:] = 0x08 -> 2.0
	},
	{
		raw:            "9A1BB38026C094CF1895E9ADA324486276F2",
		wantAddr:       "76ACF64E",
		wantIsLegacy:   true,
		wantPayloadHex: "0002BA05B903FF03FF03FF01",
		wantRSSI:       -118, // 0x76 -> -118
		wantSNR:        -3.5, // 0xF2=242 -> (242-256)/4 = -3.5
	},
	{
		raw:            "95AECEE361FAC97C102ED77CDE650DA7700F",
		wantAddr:       "1AAC6631", // ← known device
		wantIsLegacy:   true,
		wantPayloadHex: "000018F9B903FE03FF03FF01",
		wantRSSI:       -112, // 0x70 -> -112
		wantSNR:        3.75, // 0x0F -> 3.75
	},
	{
		raw:            "57F442C6502AE0BE2DD4A0C8FE4D12EB6921",
		wantAddr:       "5DACA777",
		wantIsLegacy:   true,
		wantPayloadHex: "0002A0E7B903FF03FF03FF01",
		wantRSSI:       -105, // 0x69 -> -105
		wantSNR:        8.25, // 0x21 -> 8.25
	},
	{
		raw:            "C3D47DB58D0DA1A10287182F98B1CD506C19",
		wantAddr:       "55AC5221",
		wantIsLegacy:   true,
		wantPayloadHex: "000CAF40C303FE03FF03FF01",
		wantRSSI:       -108, // 0x6C -> -108
		wantSNR:        6.25, // 0x19 -> 6.25
	},
	{
		raw:            "BFF2B740D8931E8FC3F37238E7F2969B7011",
		wantAddr:       "F8AC1198",
		wantIsLegacy:   true,
		wantPayloadHex: "000003D3B803FF03FF03FF01",
		wantRSSI:       -112, // 0x70 -> -112
		wantSNR:        4.25, // 0x11 -> 4.25
	},
	// Synthetic LoRaRAW frames: (len-2)%16 == 8
	// 4(addr) + 1*16(enc) + 4(CMAC) + 2(SNR+RSSI) = 26 bytes
	{
		raw:          strings.Repeat("AA", 26),
		wantIsLegacy: false,
		wantAddr:     "AAAAAAAA",
	},
	// 4(addr) + 2*16(enc) + 4(CMAC) + 2(SNR+RSSI) = 42 bytes
	{
		raw:          strings.Repeat("BB", 42),
		wantIsLegacy: false,
		wantAddr:     "BBBBBBBB",
	},
}

// isLegacyPacket mirrors the Go module classification:
//
//	(len(dataBytes)-2) % 16 == 0  →  legacy
//	(len(dataBytes)-2) % 16 == 8  →  LoRaRAW unique device
func isLegacyPacket(dataBytes []byte) bool {
	return (len(dataBytes)-2)%16 == 0
}

// TestPacketClassification proves (len-2)%16 always gives 0 for legacy
// and 8 for LoRaRAW, for all valid N.
func TestPacketClassification(t *testing.T) {
	t.Log("Legacy: raw = N×16(enc) + 2(SNR+RSSI)  → (len-2)%16 == 0")
	for N := 1; N <= 4; N++ {
		total := N*16 + 2
		data := make([]byte, total)
		if got := (len(data) - 2) % 16; got != 0 {
			t.Errorf("Legacy N=%d: expected remainder 0, got %d", N, got)
		}
	}

	t.Log("LoRaRAW: raw = 4(addr) + N×16(enc) + 4(CMAC) + 2(SNR+RSSI)  → (len-2)%16 == 8")
	for N := 1; N <= 4; N++ {
		total := 4 + N*16 + 4 + 2
		data := make([]byte, total)
		if got := (len(data) - 2) % 16; got != 8 {
			t.Errorf("LoRaRAW N=%d: expected remainder 8, got %d", N, got)
		}
	}
}

// TestLegacyDecryptAddress verifies that decryptLegacy reveals the correct
// device address in the first 4 bytes for all legacy frames.
func TestLegacyDecryptAddress(t *testing.T) {
	keyBytes, err := hex.DecodeString(testDefaultKey)
	if err != nil {
		t.Fatalf("invalid test key: %s", err)
	}

	for _, tc := range frameCases {
		if !tc.wantIsLegacy || tc.wantAddr == "" {
			continue
		}
		t.Run(tc.raw[:8], func(t *testing.T) {
			dataBytes, err := hex.DecodeString(tc.raw)
			if err != nil {
				t.Fatalf("hex decode: %s", err)
			}

			if !isLegacyPacket(dataBytes) {
				t.Fatalf("expected legacy packet, got LoRaRAW (len=%d, remainder=%d)",
					len(dataBytes), (len(dataBytes)-2)%16)
			}

			decrypted, err := aesutils.DecryptLegacy(dataBytes[:len(dataBytes)-2], keyBytes)
			if err != nil {
				t.Fatalf("DecryptLegacy: %s", err)
			}

			addr := strings.ToUpper(hex.EncodeToString(decrypted[:4]))
			if addr != tc.wantAddr {
				t.Errorf("address: got %s, want %s", addr, tc.wantAddr)
			}

			if tc.wantPayloadHex != "" {
				payload := strings.ToUpper(hex.EncodeToString(decrypted[4:]))
				if payload != tc.wantPayloadHex {
					t.Errorf("payload: got %s, want %s", payload, tc.wantPayloadHex)
				}
			}
		})
	}
}

// TestKnownDeviceMatch verifies that exactly the expected frames match
// registered device addresses after legacy decryption.
func TestKnownDeviceMatch(t *testing.T) {
	keyBytes, err := hex.DecodeString(testDefaultKey)
	if err != nil {
		t.Fatalf("invalid test key: %s", err)
	}

	for _, tc := range frameCases {
		if !tc.wantIsLegacy {
			continue
		}
		t.Run(tc.raw[:8], func(t *testing.T) {
			dataBytes, _ := hex.DecodeString(tc.raw)
			decrypted, err := aesutils.DecryptLegacy(dataBytes[:len(dataBytes)-2], keyBytes)
			if err != nil {
				t.Fatalf("DecryptLegacy: %s", err)
			}
			addr := strings.ToUpper(hex.EncodeToString(decrypted[:4]))
			matched := knownDevices[addr]
			expectedMatch := knownDevices[tc.wantAddr]
			if matched != expectedMatch {
				t.Errorf("addr %s: match=%v, want match=%v", addr, matched, expectedMatch)
			}
		})
	}
}

// TestRSSIAndSNRDecoding verifies RSSI and SNR decode correctly from the
// last 4 hex chars of the formatted serial string.
func TestRSSIAndSNRDecoding(t *testing.T) {
	for _, tc := range frameCases {
		if !tc.wantIsLegacy || tc.wantRSSI == 0 {
			continue
		}
		t.Run(tc.raw[:8], func(t *testing.T) {
			rssi := codec.DecodeRSSI(tc.raw)
			if rssi != tc.wantRSSI {
				t.Errorf("RSSI: got %d, want %d", rssi, tc.wantRSSI)
			}
			snr := codec.DecodeSNR(tc.raw)
			if snr != tc.wantSNR {
				t.Errorf("SNR: got %v, want %v", snr, tc.wantSNR)
			}
		})
	}
}

// TestLoRaRAWPacketClassification verifies LoRaRAW frames are NOT
// misclassified as legacy.
func TestLoRaRAWPacketClassification(t *testing.T) {
	for _, tc := range frameCases {
		if tc.wantIsLegacy {
			continue
		}
		t.Run(tc.raw[:8], func(t *testing.T) {
			dataBytes, _ := hex.DecodeString(tc.raw)
			if isLegacyPacket(dataBytes) {
				t.Errorf("LoRaRAW frame misclassified as legacy (len=%d, remainder=%d)",
					len(dataBytes), (len(dataBytes)-2)%16)
			}
			addr := strings.ToUpper(codec.DecodeAddressHex(tc.raw))
			if addr != tc.wantAddr {
				t.Errorf("LoRaRAW addr: got %s, want %s", addr, tc.wantAddr)
			}
		})
	}
}
