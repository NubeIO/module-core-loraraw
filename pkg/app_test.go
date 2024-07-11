package pkg

import (
	"sync"
	"testing"

	"github.com/NubeIO/lib-module-go/nmodule"
	"github.com/NubeIO/module-core-loraraw/decoder"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
	log "github.com/sirupsen/logrus"
)

type MockModule struct {
	*Module // Embedding Module to inherit methods

	MockDevice *model.Device // Mock device for testing
}

func (m *MockModule) handleSerialPayload(data string) {
	m.Module.handleSerialPayload(data)
}

func (m *MockModule) getDeviceByLoRaAddress(address string) *model.Device {
	// Implement mock logic to return mock device or nil based on address
	return m.MockDevice // Mock implementation, replace with your logic
}

func NewMockModule(networkUUID string, grpcMarshaller nmodule.Marshaller) *MockModule {
	return &MockModule{
		Module: &Module{
			networkUUID:    networkUUID,
			grpcMarshaller: grpcMarshaller,
			interruptChan:  make(chan struct{}),
			mutex:          &sync.RWMutex{},
		},
	}
}

func handleSerialPayload(m *Module, data string, getDeviceFn func(string) *model.Device) {
	if m.networkUUID == "" {
		return
	}
	if !decoder.ValidPayload(data) {
		return
	}
	log.Debugf("uplink: %s", data)
	device := getDeviceFn(decoder.DecodeAddress(data))
	if device == nil {
		id := decoder.DecodeAddress(data)
		rssi := decoder.DecodeRSSI(data)
		log.Infof("message from non-added sensor. ID: %s, RSSI: %d", id, rssi)
		return
	}
	devDesc := decoder.GetDeviceDescription(device)
	if devDesc == &decoder.NilLoRaDeviceDescription {
		return
	}
	err := decoder.DecodePayload(data, devDesc, device)
	if err != nil {
		log.Errorf(err.Error())
	}
}

func TestHandleSerialPayload(t *testing.T) {
	mockDevice := &model.Device{
		CommonDevice: model.CommonDevice{
			Model: "Rubix", // Initialize the nested CommonDevice struct
		},
	}

	mockModule := NewMockModule("Rubix", nil) // Provide mock marshaller if needed
	mockModule.MockDevice = mockDevice

	tests := []struct {
		name string
		data string
	}{
		{"TestValidPayload", "5CC08E7B0003B2010b04d5e0605106068600181c243c5004d21018223c762444616e28849b9bcbaf38199b89b1e609e7b837f7a1200d8085878a561a205e30a78878fffde19322a6c5cec319421c5b999c6508716e8f719421c5bae1c6508716ee1519421c5bc2946508716f33519421c5bd70c6508716f85319421c5beb8c6508716fd7119421c3fa3dc6508710e8f719421c47a3dc6508712e8f719421c4fa3dc6508714e8f719421c57a3dc6508716e8f719421c5fa3d80422A"},
		{"TestValidPayload", "5CC08E7B00018B0004D5E85106060018245004D2183C76446128849B8BAF19A1B1E5E7B7F7A00D80878A56205E30A788FFFDE19326CEC3421C5B999D08716E8F7421C5BAE1D08716EE15421C5BC29508716F335421C5BD70D08716F853421C5BEB8D08716FD71421C3FA3DD08710E8F7421C47A3DD08712E8F7421C4FA3DD08714E8F7421C57A3DD08716E8F7421C5FA3D804629"},
		// Add more test cases as needed
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call handleSerialPayload passing in the mockModule.Module and its getDeviceByLoRaAddress method
			handleSerialPayload(mockModule.Module, tt.data, mockModule.getDeviceByLoRaAddress)
			// Add assertions as needed
		})
	}
}
