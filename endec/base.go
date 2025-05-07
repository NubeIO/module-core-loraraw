package endec

import (
	"encoding/hex"
	"strconv"
)

const (
	RssiField = "rssi"
	SnrField  = "snr"
)

type CommonValues struct {
	Sensor string  `json:"sensor"`
	ID     string  `json:"id"`
	Rssi   int     `json:"rssi"`
	Snr    float32 `json:"snr"`
}

func GetCommonValueNames() []string {
	return []string{
		RssiField,
		SnrField,
	}
}

func ValidPayload(data string) bool {
	return !(len(data) <= 8)
}

func DecodeAddressHex(data string) string {
	return data[:8]
}

func DecodeAddressBytes(data []byte) string {
	return hex.EncodeToString(data[:4])
}

func DecodeRSSI(data string) int {
	dataLen := len(data)
	v, _ := strconv.ParseInt(data[dataLen-4:dataLen-2], 16, 0)
	v = v * -1
	return int(v)
}

func DecodeSNR(data string) float32 {
	dataLen := len(data)
	v, _ := strconv.ParseInt(data[dataLen-2:], 16, 0)
	var f float32
	if v > 127 {
		f = float32(v - 256)
	} else {
		f = float32(v) / 4.
	}
	return f
}
