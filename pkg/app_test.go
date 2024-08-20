package pkg

import (
	"testing"

	"github.com/NubeIO/module-core-loraraw/decoder"
	"github.com/NubeIO/module-core-loraraw/utils"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
	log "github.com/sirupsen/logrus"
)

func updateDevicePointMock(name string, value float64, device *model.Device) error {
	log.Infof("writing device %s, point %s, value: %f", device.Name, name, value)
	return nil
}

func handleSerialPayload(data string, device *model.Device) {
	if !decoder.ValidPayload(data) {
		return
	}

	devDesc := decoder.GetDeviceDescription(device)
	if devDesc == &decoder.NilLoRaDeviceDescription {
		log.Errorln("nil device description found")
		return
	}

	dataLen := len(data)
	originalData := data
	expectedMod := decoder.LoraRawHeaderLen + decoder.RssiLen + decoder.SnrLen
	if (dataLen/2)%16 == expectedMod {
		if !utils.CheckLoRaRAWPayloadLength(data) {
			log.Errorln("LoRaRaw payload length mismatched")
			return
		}
		data = utils.StripLoRaRAWPayload(data)
	}

	if !devDesc.CheckLength(data) {
		log.Errorln("invalid payload")
		return
	}

	err := decoder.DecodePayload(data, devDesc, device, updateDevicePointMock)
	if err != nil {
		log.Errorf(err.Error())
		return
	}

	rssi := decoder.DecodeRSSI(originalData)
	snr := decoder.DecodeSNR(originalData)

	_ = updateDevicePointMock(decoder.RssiField, float64(rssi), device)
	_ = updateDevicePointMock(decoder.SnrField, float64(snr), device)
}

func TestHandleSerialPayload(t *testing.T) {
	mockDevice := &model.Device{
		CommonDevice: model.CommonDevice{
			Model: "Rubix", // Initialize the nested CommonDevice struct
		},
	}

	tests := []struct {
		name string
		data string
	}{
		{"TestValidPayload", "5CC08E7B0006B2010B04D5E8605106068600181C243C5004D21018223C762444616E28849B9BCBAF3819A389B1E609E7B837F7A1200D8085878A561A205E30A78878FFFDE19322A6C5CEC319421C5B999C6D08716E8F71D421C5BAE1C7D08716EE1521421C5BC2948D08716F33525421C5BD70C9D08716F85329421C5BEB8CAD08716FD712D421C3FA3DCBD08710E8F731421C47A3DCCD08712E8F735421C4FA3DCDD08714E8F739421C57A3DCED08716E8F73D421C5FA3D80A22951681CA069A2463A43412A"},
		// LORALOAD=11+1+23.45        # temp
		// LORALOAD=12+2+87.16        # humi
		// LORALOAD=13+3+12           # lux
		// LORALOAD=14+4+1            # movement
		// LORALOAD=15+5+1234         # Pulses/counter
		// LORALOAD=16+6+0            # Digital
		// LORALOAD=17+7+5.71         # 0-10V
		// LORALOAD=18+8+15.21        # 4-20mA
		// LORALOAD=110+10+135790     # Ohm
		// LORALOAD=111+11+350        # CO2
		// LORALOAD=112+12+5.2        # Battery Voltage
		// LORALOAD=113+13+1145       # Push Frequency
		// LORALOAD=130+30+123        # uint8
		// LORALOAD=131+31+-34        # int8
		// LORALOAD=132+32+3456       # uint16
		// LORALOAD=133+33+-7531      # int16
		// LORALOAD=134+34+98765432   # uint32
		// LORALOAD=135+35+-555444    # int32
		// LORALOAD=138+38+1          # bool
		// LORALOAD=139+39+a          # char
		// LORALOAD=140+40+278.90     # float
		// LORALOAD=141+40+278.91     # float
		// LORALOAD=142+40+278.92     # float
		// LORALOAD=143+40+278.93     # float
		// LORALOAD=144+40+278.94     # float
		// LORALOAD=145+40+278.95     # float
		// LORALOAD=146+40+278.96     # float
		// LORALOAD=147+40+278.97     # float
		// LORALOAD=148+40+278.98     # float
		// LORALOAD=149+40+278.99     # float
		// LORALOAD=150+40+271.91     # float
		// LORALOAD=151+40+272.91     # float
		// LORALOAD=152+40+273.91     # float
		// LORALOAD=153+40+274.91     # float
		// LORALOAD=154+40+275.91     # float
		// LORALOAD=155+40+276.91     # float
		// LORALOAD=156+40+277.91     # float
		// LORALOAD=157+40+278.91     # float
		// LORALOAD=158+40+279.91     # float
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call handleSerialPayload passing in the mockModule.Module and its getDeviceByLoRaAddress method
			handleSerialPayload(tt.data, mockDevice)
			// Add assertions as needed
		})
	}
}

func TestMicroEdgePayload(t *testing.T) {
	mockDevice := &model.Device{
		Name: "MicroEdge",
		CommonDevice: model.CommonDevice{
			Model: "MicroEdgeV1",
		},
	}

	tests := []struct {
		name string
		data string
	}{
		{"MicroEdgeOne", "17AC7BB100000000FF03FF03FF03FF014B5F"},
		// Type:             Micro Edge
		// Node ID          : 17AC7BB1
		// Analog 1         : 1023
		// Analog 2         : 1023
		// Analog 3         : 1023
		// Pulses           : 0
		// Voltage          : 5.1
		{"MicroEdgeTwo", "55ACA79B00000000FF03FF03FF03FF013F64"},
		// Type:             Micro Edge
		// Node ID          : 55ACA79B
		// Analog 1         : 1023
		// Analog 2         : 1023
		// Analog 3         : 1023
		// Pulses           : 0
		// Voltage          : 5.1
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handleSerialPayload(tt.data, mockDevice)
		})
	}
}

func TestDropletPayload(t *testing.T) {
	mockDevice := &model.Device{
		Name: "Droplet",
		CommonDevice: model.CommonDevice{
			Model: "THLM",
		},
	}

	tests := []struct {
		name string
		data string
	}{
		{"DropletOne", "CBB272EAB20696263C0000DD000000041861"},
		// Type:         Droplet
		// Temperature:  17.14˚C
		// Pressure:     987.80hPa
		// Humidity:     60%
		// Movement:     false
		// Light:        0 lux
		// Voltage:      4.42v
		{"DropletTwo", "1AB22D4F2006C6263D0200DB000000E81A61"},
		// Type:         Droplet
		// Temperature:  15.68˚C
		// Pressure:     992.60hPa
		// Humidity:     61%
		// Movement:     false
		// Light:        2 lux
		// Voltage:      4.38v
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handleSerialPayload(tt.data, mockDevice)
		})
	}
}
