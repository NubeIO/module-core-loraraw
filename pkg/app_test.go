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

	if !devDesc.CheckLength(data) {
		log.Errorln("invalid payload")
		return
	}

	dataLen := len(data)
	originalData := data
	expectedMod := decoder.LoraRawHeaderLen + decoder.LoraRawCmacLen + decoder.RssiLen + decoder.SnrLen
	if (dataLen/2)%16 == expectedMod {
		if !utils.CheckLoRaRAWPayloadLength(data) {
			log.Errorln("LoRaRaw payload length mismatched")
			return
		}

		data = data[14:utils.GetInnerPayloadLength(data)]
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
		// {"TestValidPayload", "5CC08E7B0003B2010b04d5e0605106068600181c243c5004d21018223c762444616e28849b9bcbaf38199b89b1e609e7b837f7a1200d8085878a561a205e30a78878fffde19322a6c5cec319421c5b999c6508716e8f719421c5bae1c6508716ee1519421c5bc2946508716f33519421c5bd70c6508716f85319421c5beb8c6508716fd7119421c3fa3dc6508710e8f719421c47a3dc6508712e8f719421c4fa3dc6508714e8f719421c57a3dc6508716e8f719421c5fa3d80422A"},
		// {"TestValidPayload", "5CC08E7B00018B0004D5E85106060018245004D2183C76446128849B8BAF19A1B1E5E7B7F7A00D80878A56205E30A788FFFDE19326CEC3421C5B999D08716E8F7421C5BAE1D08716EE15421C5BC29508716F335421C5BD70D08716F853421C5BEB8D08716FD71421C3FA3DD08710E8F7421C47A3DD08712E8F7421C4FA3DD08714E8F7421C57A3DD08716E8F7421C5FA3D804629"},
		{"TestValidPayload", "5CC08E7B0006B2010B04D5E8605106068600181C243C5004D21018223C762444616E28849B9BCBAF3819A389B1E609E7B837F7A1200D8085878A561A205E30A78878FFFDE19322A6C5CEC319421C5B999C6D08716E8F71D421C5BAE1C7D08716EE1521421C5BC2948D08716F33525421C5BD70C9D08716F85329421C5BEB8CAD08716FD712D421C3FA3DCBD08710E8F731421C47A3DCCD08712E8F735421C4FA3DCDD08714E8F739421C57A3DCED08716E8F73D421C5FA3D80A22951681CA069A2463A43B6D1F892412A"},
		// Add more test cases as needed
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call handleSerialPayload passing in the mockModule.Module and its getDeviceByLoRaAddress method
			handleSerialPayload(tt.data, mockDevice)
			// Add assertions as needed
		})
	}
}
