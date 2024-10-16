package pkg

import (
	"fmt"
	"testing"

	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
	log "github.com/sirupsen/logrus"
)

func runEncryptedTests(tests []TestStruct, mockDevice *model.Device, t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			currTest = &tt
			currIndex = 0
			fmt.Printf("TEST %s\r\n", tt.Name)
			dataLegacy, err := decryptLegacy(tt.Data, "0301021604050f07e6095a0b0c12630f")
			if err == nil {
				tt.Data = dataLegacy
			}
			err = decodeData(tt.Data, mockDevice, updateDevicePointMock, updateDeviceMetaTagsMock)
			if err != nil {
				log.Errorf("decode error: %v\r\n", err)
			}
		})
	}
}

func runEncryptedRubixTests(tests []TestStruct, mockDevice *model.Device, t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			currTest = &tt
			currIndex = 0
			dataLegacy, err := decryptNormal(tt.Data, "0301021604050F07E6095A0B0C12630F")
			if err != nil {
				log.Errorf("error decrypting data: %s", err)
			}
			err = decodeData(dataLegacy, mockDevice, updateDevicePointMock, updateDeviceMetaTagsMock)
			if err != nil {
				log.Errorf("error decode data: %s", err)
			}
		})
	}
}

func TestMicroEdge1Payload(t *testing.T) {
	test = t
	mockDevice := &model.Device{
		Name: "MicroEdge",
		CommonDevice: model.CommonDevice{
			Model: "MicroEdgeV1",
		},
	}

	tests := []TestStruct{
		{"MicroEdgeOne",
			"75F3BCB93FEDA4925D1946BE5C676A0A5525",
			[]TestPoint{
				{"pulse", 0.000000},
				{"voltage", 3.680000},
				{"ai_1", 1023.000000},
				{"ai_2", 1023.000000},
				{"ai_3", 1023.000000},
			},
			[]*model.DeviceMetaTag{},
		},
		{"MicroEdgeTwo",
			"F2B70465BA3D399B73F82D2658EF563B5A28",
			[]TestPoint{
				{"pulse", 0.000000},
				{"voltage", 3.680000},
				{"ai_1", 1023.000000},
				{"ai_2", 1023.000000},
				{"ai_3", 1023.000000},
			},
			[]*model.DeviceMetaTag{},
		},
	}

	runEncryptedTests(tests, mockDevice, t)
}

func TestEncryotedDropletPayload(t *testing.T) {
	test = t
	mockDevice := &model.Device{
		Name: "Droplet",
		CommonDevice: model.CommonDevice{
			Model: "THLM",
		},
	}

	tests := []TestStruct{
		{"DropletOne",
			"C4CFE357A76AEAC8A387A9FDB69BDD0B3F28",
			[]TestPoint{
				{"temperature", 24.280000},
				{"pressure", 986.700000},
				{"humidity", 37.000000},
				{"voltage", 4.700000},
				{"light", 0.000000},
				{"motion", 0.000000},
			},
			[]*model.DeviceMetaTag{},
		},
		{"DropletTwo",
			"8DDBBBF7F3C0DDFEB4BC0FEB6CECDFC5352B",
			[]TestPoint{
				{"temperature", 20.770000},
				{"pressure", 987.300000},
				{"humidity", 48.000000},
				{"voltage", 4.720000},
				{"light", 0.000000},
				{"motion", 0.000000},
			},
			[]*model.DeviceMetaTag{},
		},
	}

	runEncryptedTests(tests, mockDevice, t)
}

func TestEncryptedRubixPayload(t *testing.T) {
	test = t
	mockDevice := &model.Device{
		Name: "Rubix",
		CommonDevice: model.CommonDevice{
			Model: "Rubix",
		},
	}

	tests := []TestStruct{
		{"RubixOne",
			"5CC08E7B0547C75CF319679F441CE5E245791166007507AD486373E723865A51C914B42EDC256D94120EBA8CAE0C26BC7B23CBC57ADE51C34A26",
			[]TestPoint{
				{"bool-1", 0.000000},
				{"uint_8-2", 0.000000},
				{"temp-3", 18.000000},
				{"uint_8-4", 0.000000},
				{"temp-5", 18.000000},
				{"uint_16-6", 0.000000},
				{"uint_16-7", 0.000000},
				{"uint_16-8", 0.000000},
				{"uint_16-9", 0.000000},
			},
			[]*model.DeviceMetaTag{},
		},
		{"RubixTwo",
			"5CC08E7B3F48B2EC9F8BCD7C0B086593E2627266DC4AB1406448C223128DCBCC87B2AF3AEA5661B9D059AD2D5D8948CF782EAF8AD00EA2BE4928",
			[]TestPoint{
				{"bool-1", 0.000000},
				{"uint_8-2", 1.000000},
				{"temp-3", 18.000000},
				{"uint_8-4", 2.000000},
				{"temp-5", 18.000000},
				{"uint_16-6", 0.000000},
				{"uint_16-7", 0.000000},
				{"uint_16-8", 0.000000},
				{"uint_16-9", 0.000000},
			},
			[]*model.DeviceMetaTag{},
		},
	}

	runEncryptedRubixTests(tests, mockDevice, t)
}
