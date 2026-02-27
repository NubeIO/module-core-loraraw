package codec

import (
	"encoding/hex"
	"errors"
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

func DecodeAddressHex(data string) (string, error) {
	if len(data) < 8 {
		return "", errors.New("data too short to decode address: required=8 actual=" + strconv.Itoa(len(data)))
	}
	return data[:8], nil
}

func DecodeAddressBytes(data []byte) (string, error) {
	if len(data) < 4 {
		return "", errors.New("data too short to decode address: required=4 actual=" + strconv.Itoa(len(data)))
	}
	return hex.EncodeToString(data[:4]), nil
}

func DecodeRSSI(data string) (int, error) {
	dataLen := len(data)
	if dataLen < 4 {
		return 0, errors.New("data too short to decode RSSI: required=4 actual=" + strconv.Itoa(dataLen))
	}
	v, err := strconv.ParseInt(data[dataLen-4:dataLen-2], 16, 0)
	if err != nil {
		return 0, errors.New("failed to parse RSSI value: " + err.Error())
	}
	v = v * -1
	return int(v), nil
}

func DecodeSNR(data string) (float32, error) {
	dataLen := len(data)
	if dataLen < 2 {
		return 0, errors.New("data too short to decode SNR: required=2 actual=" + strconv.Itoa(dataLen))
	}
	v, err := strconv.ParseInt(data[dataLen-2:], 16, 0)
	if err != nil {
		return 0, errors.New("failed to parse SNR value: " + err.Error())
	}
	var f float32
	if v > 127 {
		f = float32(v - 256)
	} else {
		f = float32(v) / 4.
	}
	return f, nil
}
