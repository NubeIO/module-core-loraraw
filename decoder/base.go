package decoder

import (
	"errors"
	"github.com/NubeIO/lib-module-go/nmodule"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
	"strconv"
)

var grpcMarshaller nmodule.Marshaller

const (
	SensorField = "sensor"
	IDField     = "id"
	RssiField   = "rssi"
	SnrField    = "snr"
)

type CommonValues struct {
	Sensor string  `json:"sensor"`
	ID     string  `json:"id"`
	Rssi   int     `json:"rssi"`
	Snr    float32 `json:"snr"`
}

func InitGrpcMarshaller(marshaller nmodule.Marshaller) {
	grpcMarshaller = marshaller
}

func GetCommonValueNames() []string {
	return []string{
		RssiField,
		SnrField,
	}
}

func DecodePayload(data string, devDesc *LoRaDeviceDescription, device *model.Device) error {
	if !devDesc.CheckLength(data) {
		return errors.New("invalid payload")
	}
	err := devDesc.Decode(data, devDesc, device)
	return err
}

func ValidPayload(data string) bool {
	return !(len(data) <= 8)
}

func DecodeAddress(data string) string {
	return data[:8]
}

func decodeCommonValues(payload *CommonValues, data string, sensor string) {
	payload.Sensor = sensor
	payload.ID = DecodeAddress(data)
	payload.Rssi = DecodeRSSI(data)
	payload.Snr = decodeSNR(data)
}

func DecodeRSSI(data string) int {
	dataLen := len(data)
	v, _ := strconv.ParseInt(data[dataLen-4:dataLen-2], 16, 0)
	v = v * -1
	return int(v)
}

func decodeSNR(data string) float32 {
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
