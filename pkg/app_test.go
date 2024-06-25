package pkg

import (
	"fmt"
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
	commonData, fullData := decoder.DecodePayload(data, devDesc)
	if commonData == nil {
		return
	}
	log.Infof("sensor found. ID: %s, RSSI: %d, Type: %s", commonData.ID, commonData.Rssi, commonData.Sensor)
	// _ = m.grpcMarshaller.UpdateDeviceFault(device.UUID, &model.CommonFault{
	// 	InFault: false,
	// 	Message: "",
	// })
	// if fullData != nil {
	// 	m.updateDevicePointValues(commonData, fullData, device)
	// }
	fmt.Println(fullData)
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
		{"TestValidPayload", "8AC03117005A00055cf04ad986cd2c24530d3f1a3ef044c029b2070ba698e9a03d0de111912321329f45be2c7941fb08019061f9bf88001388063fffb1dff900000003b9aca00065fffffff1194d7fff9d8683f8fbe77a2fe3ef9de84479c05248014427"},
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
