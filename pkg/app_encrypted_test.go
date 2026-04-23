package pkg

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/NubeIO/module-core-loraraw/codec"
	"github.com/NubeIO/module-core-loraraw/schema"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
	log "github.com/sirupsen/logrus"
)

func runEncryptedTests(tests []TestStruct, mockDevice *model.Device, t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			currTest = &tt
			currIndex = 0
			fmt.Printf("TEST %s\r\n", tt.Name)
			dataBytes, err := hex.DecodeString(tt.Data)
			if err != nil {
				log.Errorf("hex decode error: %s", err)
				return
			}
			keyBytes, err := hex.DecodeString("0301021604050f07e6095a0b0c12630f")
			if err != nil {
				log.Errorf("hex decode key error: %s", err)
				return
			}
			dataLegacy, err := decryptLegacy(dataBytes, keyBytes)
			if err == nil {
				tt.Data = hex.EncodeToString(dataLegacy)
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
			dataBytes, err := hex.DecodeString(tt.Data)
			if err != nil {
				log.Errorf("hex decode error: %s", err)
				return
			}
			keyBytes, err := hex.DecodeString("0301021604050F07E6095A0B0C12630F")
			if err != nil {
				log.Errorf("hex decode key error: %s", err)
				return
			}
			decrypted, err := decryptLoRaRAWPkt(dataBytes, keyBytes)
			if err != nil {
				log.Errorf("error decrypting data: %s", err)
				return
			}
			err = decodeData(hex.EncodeToString(decrypted), mockDevice, updateDevicePointMock, updateDeviceMetaTagsMock)
			if err != nil {
				log.Errorf("error decode data: %s", err)
			}
		})
	}
}

// runEncryptedZHTTests decrypts each raw LoRaRAW frame with the shared test
// key, then dispatches the inner payload through the same code path the live
// pipeline uses (decodeData -> DecodeZHT). The test fails if decryption or
// decoding reports an error, guarding against regressions of the ZHT decoder
// (e.g. decoding the raw address instead of the stripped payload).
//
// All decoded points are logged so fixtures can be extended with exact values
// in the future without touching the runner.
func runEncryptedZHTTests(tests []TestStruct, mockDevice *model.Device, t *testing.T) {
	keyBytes, err := hex.DecodeString("0301021604050F07E6095A0B0C12630F")
	if err != nil {
		t.Fatalf("hex decode key error: %s", err)
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			currTest = &tt
			currIndex = 0

			dataBytes, err := hex.DecodeString(tt.Data)
			if err != nil {
				t.Fatalf("hex decode error: %s", err)
			}
			decrypted, err := decryptLoRaRAWPkt(dataBytes, keyBytes)
			if err != nil {
				t.Fatalf("decryptLoRaRAWPkt failed: %s", err)
			}

			capture := func(name string, value float64, device *model.Device, devDesc *codec.LoRaDeviceDescription) error {
				t.Logf("  %s = %f", name, value)
				return updateDevicePointMock(name, value, device, devDesc)
			}

			err = decodeData(hex.EncodeToString(decrypted), mockDevice, capture, updateDeviceMetaTagsMock)
			if err != nil {
				t.Fatalf("decodeData failed: %s", err)
			}
		})
	}
}

// TestMicroEdge1Payload ...
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

// TestEncryotedDropletPayload ...
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

// TestEncryptedRubixPayload ...
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
				{"unknown-1", 0.000000},
				{"UI-7", 0.000000},
				{"DO-16", 0.000000},
				{"unknown-1", 0.000000},
				{"UO-7", 0.000000},
				{"UVP-33", 0.000000},
				{"DVP-1", 0.000000},
				{"movement-21", 1.000000},
				{"DI-10", 0.000000},
				{"unknown-27", 0.000000},
				{"unknown-1", 0.000000},
				{"uint_16-8", 0.000000},
				{"uint_16-9", 0.000000},
				{"uint_16-10", 0.000000},
			},
			[]*model.DeviceMetaTag{},
		},
		{"RubixTwo",
			"5CC08E7B3F48B2EC9F8BCD7C0B086593E2627266DC4AB1406448C223128DCBCC87B2AF3AEA5661B9D059AD2D5D8948CF782EAF8AD00EA2BE4928",
			[]TestPoint{
				{"unknown-1", 0.000000},
				{"UI-7", 0.000000},
				{"DO-16", 0.000000},
				{"UO-1", 0.000000},
				{"UO-7", 0.000000},
				{"UVP-33", 0.000000},
				{"DVP-1", 5.600000},
				{"DI-25", 1.000000},
				{"UVP-1", 0.000000},
				{"unknown-1", 0.000000},
				{"UI-25", 0.000000},
				{"unknown-1", 0.000000},
				{"UO-3", 0.000000},
				{"unknown-1", 0.000000},
				{"uint_16-10", 0.000000},
			},
			[]*model.DeviceMetaTag{},
		},
	}

	runEncryptedRubixTests(tests, mockDevice, t)
}

// TestEncryptedZHTPayload replays real encrypted LoRaRAW frames captured from
// a ZipHydroTap (address 00C032AA). It verifies the full pipeline:
//
//	raw hex -> decryptLoRaRAWPkt -> StripLoRaRAWPayload -> DecodeZHT
//
// Historically DecodeZHT consumed the raw data string (which starts with the
// 4-byte address), making the first byte (0x00) parse as ErrorData and the
// decoder a silent no-op. These fixtures keep that regression from returning.
func TestEncryptedZHTPayload(t *testing.T) {
	test = t
	addr := "00C032AA"
	mockDevice := &model.Device{
		Name: "ZHT",
		CommonDevice: model.CommonDevice{
			Model:       schema.DeviceModelZiptHydroTap,
			AddressUUID: &addr,
		},
	}

	tests := []TestStruct{
		{
			Name: "ZHT-Encrypted-1",
			Data: "00C032AAB0138AB28B6E9A969E7E9CCCA2032EA05837EDF19D35014D38697EB48F591B05E27C93089C3B6A6AF567CA517EAB07A8D8FB11A772C7B1310ABA061D8C6E933163A5AD085228",
			Values: []TestPoint{
				{"rebooted", 0}, {"sleep_mode_status", 1},
				{"temperature_ntc_boiling", 98.0}, {"temperature_ntc_chilled", 8.4},
				{"temperature_ntc_stream", 61.0}, {"temperature_ntc_condensor", 30.0},
				{"fault_1", 255}, {"fault_2", 255}, {"fault_3", 255}, {"fault_4", 255},
				{"usage_energy_kwh", 4598.600098},
				{"usage_water_delta_dispenses_boiling", 0}, {"usage_water_delta_dispenses_chilled", 0}, {"usage_water_delta_dispenses_sparkling", 0},
				{"usage_water_delta_litres_boiling", 0}, {"usage_water_delta_litres_chilled", 0}, {"usage_water_delta_litres_sparkling", 0},
				{"filter_warning_internal", 0}, {"filter_warning_external", 0},
				{"filter_info_usage_litres_internal", 700}, {"filter_info_usage_days_internal", 160},
				{"filter_info_usage_litres_external", 0}, {"filter_info_usage_days_external", 0},
				{"filter_info_usage_litres_uv", 0}, {"filter_info_usage_days_uv", 0},
				{"filter_warning_uv", 0}, {"co2_low_gas_warning", 0},
				{"co2_usage_grams", 0}, {"co2_usage_days", 0},
			},
			MetaTags: []*model.DeviceMetaTag{},
		},
		{
			Name: "ZHT-Encrypted-2",
			Data: "00C032AA7BDB7ECADE4042FDB93280AFCD41B002E530E1A0FBEEB0B51FE48E78463B98BB815CE57D36DF5563F38E330A9EF56A611D1A68B1DA6327D95647896B9C567A499CBC55315228",
			Values: []TestPoint{
				{"rebooted", 0}, {"sleep_mode_status", 1},
				{"temperature_ntc_boiling", 97.5}, {"temperature_ntc_chilled", 8.4},
				{"temperature_ntc_stream", 61.0}, {"temperature_ntc_condensor", 30.0},
				{"fault_1", 255}, {"fault_2", 255}, {"fault_3", 255}, {"fault_4", 255},
				{"usage_energy_kwh", 4598.600098},
				{"usage_water_delta_dispenses_boiling", 0}, {"usage_water_delta_dispenses_chilled", 0}, {"usage_water_delta_dispenses_sparkling", 0},
				{"usage_water_delta_litres_boiling", 0}, {"usage_water_delta_litres_chilled", 0}, {"usage_water_delta_litres_sparkling", 0},
				{"filter_warning_internal", 0}, {"filter_warning_external", 0},
				{"filter_info_usage_litres_internal", 700}, {"filter_info_usage_days_internal", 160},
				{"filter_info_usage_litres_external", 0}, {"filter_info_usage_days_external", 0},
				{"filter_info_usage_litres_uv", 0}, {"filter_info_usage_days_uv", 0},
				{"filter_warning_uv", 0}, {"co2_low_gas_warning", 0},
				{"co2_usage_grams", 0}, {"co2_usage_days", 0},
			},
			MetaTags: []*model.DeviceMetaTag{},
		},
		{
			Name: "ZHT-Encrypted-3",
			Data: "00C032AADF0105FE4A892F43F1B6490450A54EFDA8721323CC19AD5AB478054B2D47573EA58DF06E07E5A8EB650DFFFC231C863261D3173174348EB6CD36D160D775FF796C6A56435228",
			Values: []TestPoint{
				{"rebooted", 0}, {"sleep_mode_status", 1},
				{"temperature_ntc_boiling", 98.0}, {"temperature_ntc_chilled", 8.4},
				{"temperature_ntc_stream", 60.5}, {"temperature_ntc_condensor", 30.0},
				{"fault_1", 255}, {"fault_2", 255}, {"fault_3", 255}, {"fault_4", 255},
				{"usage_energy_kwh", 4598.600098},
				{"usage_water_delta_dispenses_boiling", 0}, {"usage_water_delta_dispenses_chilled", 0}, {"usage_water_delta_dispenses_sparkling", 0},
				{"usage_water_delta_litres_boiling", 0}, {"usage_water_delta_litres_chilled", 0}, {"usage_water_delta_litres_sparkling", 0},
				{"filter_warning_internal", 0}, {"filter_warning_external", 0},
				{"filter_info_usage_litres_internal", 700}, {"filter_info_usage_days_internal", 160},
				{"filter_info_usage_litres_external", 0}, {"filter_info_usage_days_external", 0},
				{"filter_info_usage_litres_uv", 0}, {"filter_info_usage_days_uv", 0},
				{"filter_warning_uv", 0}, {"co2_low_gas_warning", 0},
				{"co2_usage_grams", 0}, {"co2_usage_days", 0},
			},
			MetaTags: []*model.DeviceMetaTag{},
		},
		{
			Name: "ZHT-Encrypted-4",
			Data: "00C032AAA47E9A05A2AD7A45B19F886F1F05777F3B6C6AFABA2706EFB310B85D4EDB4C4E4A82791DB2D92B49006E3F9A5095C09F24D0DF98FD657D815F6FC406CBD2E93096766AA05228",
			Values: []TestPoint{
				{"rebooted", 0}, {"sleep_mode_status", 1},
				{"temperature_ntc_boiling", 98.0}, {"temperature_ntc_chilled", 8.4},
				{"temperature_ntc_stream", 61.0}, {"temperature_ntc_condensor", 30.0},
				{"fault_1", 255}, {"fault_2", 255}, {"fault_3", 255}, {"fault_4", 255},
				{"usage_energy_kwh", 4598.600098},
				{"usage_water_delta_dispenses_boiling", 0}, {"usage_water_delta_dispenses_chilled", 0}, {"usage_water_delta_dispenses_sparkling", 0},
				{"usage_water_delta_litres_boiling", 0}, {"usage_water_delta_litres_chilled", 0}, {"usage_water_delta_litres_sparkling", 0},
				{"filter_warning_internal", 0}, {"filter_warning_external", 0},
				{"filter_info_usage_litres_internal", 700}, {"filter_info_usage_days_internal", 160},
				{"filter_info_usage_litres_external", 0}, {"filter_info_usage_days_external", 0},
				{"filter_info_usage_litres_uv", 0}, {"filter_info_usage_days_uv", 0},
				{"filter_warning_uv", 0}, {"co2_low_gas_warning", 0},
				{"co2_usage_grams", 0}, {"co2_usage_days", 0},
			},
			MetaTags: []*model.DeviceMetaTag{},
		},
		{
			Name: "ZHT-Encrypted-5",
			Data: "00C032AA409D0A9B1067C9517BF77194AD07254AB06773F5C7B2004405B38C71F7315A6DE8C70CE9F105B1C5609D0918876AC96BE06C5848326B478850A67FB4B4AD23F9DACAC1F65227",
			Values: []TestPoint{
				{"rebooted", 0}, {"sleep_mode_status", 1},
				{"temperature_ntc_boiling", 97.5}, {"temperature_ntc_chilled", 8.4},
				{"temperature_ntc_stream", 61.0}, {"temperature_ntc_condensor", 30.4},
				{"fault_1", 255}, {"fault_2", 255}, {"fault_3", 255}, {"fault_4", 255},
				{"usage_energy_kwh", 4598.600098},
				{"usage_water_delta_dispenses_boiling", 0}, {"usage_water_delta_dispenses_chilled", 0}, {"usage_water_delta_dispenses_sparkling", 0},
				{"usage_water_delta_litres_boiling", 0}, {"usage_water_delta_litres_chilled", 0}, {"usage_water_delta_litres_sparkling", 0},
				{"filter_warning_internal", 0}, {"filter_warning_external", 0},
				{"filter_info_usage_litres_internal", 700}, {"filter_info_usage_days_internal", 160},
				{"filter_info_usage_litres_external", 0}, {"filter_info_usage_days_external", 0},
				{"filter_info_usage_litres_uv", 0}, {"filter_info_usage_days_uv", 0},
				{"filter_warning_uv", 0}, {"co2_low_gas_warning", 0},
				{"co2_usage_grams", 0}, {"co2_usage_days", 0},
			},
			MetaTags: []*model.DeviceMetaTag{},
		},
		{
			Name: "ZHT-Encrypted-6",
			Data: "00C032AA20DC48F0D608F9FE9EF46474AAA0C91BAC321A936A1FF900249B82A026CFD86E85F44AAA4C8A147138A76F333CF2404FD479928F7189D250839BE5E25B54A5D368C78826522A",
			Values: []TestPoint{
				{"rebooted", 0}, {"sleep_mode_status", 1},
				{"temperature_ntc_boiling", 98.0}, {"temperature_ntc_chilled", 8.4},
				{"temperature_ntc_stream", 61.0}, {"temperature_ntc_condensor", 30.0},
				{"fault_1", 255}, {"fault_2", 255}, {"fault_3", 255}, {"fault_4", 255},
				{"usage_energy_kwh", 4598.600098},
				{"usage_water_delta_dispenses_boiling", 0}, {"usage_water_delta_dispenses_chilled", 0}, {"usage_water_delta_dispenses_sparkling", 0},
				{"usage_water_delta_litres_boiling", 0}, {"usage_water_delta_litres_chilled", 0}, {"usage_water_delta_litres_sparkling", 0},
				{"filter_warning_internal", 0}, {"filter_warning_external", 0},
				{"filter_info_usage_litres_internal", 700}, {"filter_info_usage_days_internal", 160},
				{"filter_info_usage_litres_external", 0}, {"filter_info_usage_days_external", 0},
				{"filter_info_usage_litres_uv", 0}, {"filter_info_usage_days_uv", 0},
				{"filter_warning_uv", 0}, {"co2_low_gas_warning", 0},
				{"co2_usage_grams", 0}, {"co2_usage_days", 0},
			},
			MetaTags: []*model.DeviceMetaTag{},
		},
	}

	runEncryptedZHTTests(tests, mockDevice, t)
}
