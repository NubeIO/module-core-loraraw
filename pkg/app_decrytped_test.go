package pkg

import (
	"fmt"
	"math"
	"testing"

	"github.com/NubeIO/module-core-loraraw/schema"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
)

type TestPoint struct {
	Name  string
	Value float64
}

type TestStruct struct {
	Name     string
	Data     string
	Values   []TestPoint
	MetaTags []*model.DeviceMetaTag
}

var (
	currTest  *TestStruct
	currIndex int
	test      *testing.T
)

const float64EqualityThreshold = 1e-6

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) <= float64EqualityThreshold
}

func runTests(tests []TestStruct, mockDevice *model.Device, t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			currTest = &tt
			currIndex = 0
			fmt.Printf("TEST %s\r\n", tt.Name)
			err := decodeData(tt.Data, mockDevice, updateDevicePointMock, updateDeviceMetaTagsMock)
			if err != nil {
				test.Logf("FAILED decodeData(): %v", err)
				test.Fail()
			}
		})
	}
}

func updateDevicePointMock(name string, value float64, device *model.Device) error {
	// fmt.Printf("{\"%s\", %f},\n", name, value)
	devName := currTest.Name
	expectedName := currTest.Values[currIndex].Name
	expectedValue := currTest.Values[currIndex].Value

	if name != expectedName {
		test.Logf("FAILED NAME. Device: %s, expected: %s, actual: %s", devName, expectedName, name)
		test.Fail()
	}
	if !almostEqual(value, expectedValue) {
		test.Logf("FAILED VALUE. Device: %s, expected: %f, actual: %f", devName, expectedValue, value)
		test.Fail()
	}
	currIndex++
	return nil
}

func updateDeviceMetaTagsMock(uuid string, metaTags []*model.DeviceMetaTag) error {
	// fmt.Print("meta-tags: {")
	// for _, tag := range metaTags {
	// 	fmt.Printf("{DeviceUUID:\"%s\", Key:\"%s\", Value:\"%s\"},", uuid, tag.Key, tag.Value)
	// }
	// fmt.Println("}")

	if len(metaTags) != len(currTest.MetaTags) {
		test.Logf("FAILED TAGS LENGTH. uuid: %s, expected: %d, actual: %d", uuid, len(currTest.MetaTags), len(metaTags))
		test.Fail()
	}
	for _, tag := range metaTags {
		var expTag *model.DeviceMetaTag = nil
		for _, t := range currTest.MetaTags {
			if t.Key == tag.Key {
				expTag = t
				break
			}
		}
		if expTag == nil {
			test.Logf("FAILED UNEXPECTED TAG. uuid: %s, actualKey: %s, actualValue: %s", uuid, tag.Key, tag.Value)
			test.Fail()
		} else if tag.Key != expTag.Key || tag.Value != expTag.Value {
			test.Logf("FAILED TAG. uuid: %s, expectedKey: %s, actualKey: %s, expectedValue: %s, actualValue: %s", uuid, expTag.Key, tag.Key, expTag.Value, tag.Value)
			test.Fail()
		}
	}
	return nil
}

func TestRubixPayload(t *testing.T) {
	test = t
	mockDevice := &model.Device{
		CommonDevice: model.CommonDevice{
			Model: "Rubix",
		},
	}

	tests := []TestStruct{
		{"dummyRubix",
			// "5CC08E7B0006B2010B04D5E8605106068600181C243C5004D21018223C762444616E28849B9BCBAF3819A389B1E609E7B837F7A1200D8085878A561A205E30A78878FFFDE19322A6C5CEC319421C5B999C6D08716E8F71D421C5BAE1C7D08716EE1521421C5BC2948D08716F33525421C5BD70C9D08716F85329421C5BEB8CAD08716FD712D421C3FA3DCBD08710E8F731421C47A3DCCD08712E8F735421C4FA3DCDD08714E8F739421C57A3DCED08716E8F73D421C5FA3D80A22951681CA069A2463A43412A",
			"5CC08E7B0006B2010B04D5E8605106068600181C243C5004D21018223C762444616E28849B9BCBAF3819A389B1E609E7B837F7A1200D8085878A561A205E30A78878FFFDE19322A6C5CEC319421C5B999C6D08716E8F71D421C5BAE1C7D08716EE1521421C5BC2948D08716F33525421C5BD70C9D08716F85329421C5BEB8CAD08716FD712D421C3FA3DCBD08710E8F731421C47A3DCCD08712E8F735421C4FA3DCDD08714E8F739421C57A3DCED08716E8F73D421C5FA3D80412A",
			[]TestPoint{
				{"temp-11", 23.450001},
				{"rh-12", 87.160004},
				{"lux-13", 12.000000},
				{"movement-14", 1.000000},
				{"count-15", 1234.000000},
				{"digital-16", 0.000000},
				{"0-10v-17", 5.710000},
				{"4-20ma-18", 15.210000},
				{"ohm-110", 135790.000000},
				{"co2-111", 350.000000},
				{"battery-voltage-112", 5.200000},
				{"push-frequency-113", 1145.000000},
				{"uint_8-130", 123.000000},
				{"int_8-131", -34.000000},
				{"uint_16-132", 3456.000000},
				{"int_16-133", -7531.000000},
				{"uint_32-134", 98765432.000000},
				{"int_32-135", -555444.000000},
				{"bool-138", 1.000000},
				{"char-139", 97.000000},
				{"float-140", 278.899994},
				{"float-141", 278.910004},
				{"float-142", 278.920013},
				{"float-143", 278.929993},
				{"float-144", 278.940002},
				{"float-145", 278.950012},
				{"float-146", 278.959991},
				{"float-147", 278.970001},
				{"float-148", 278.980011},
				{"float-149", 278.989990},
				{"float-150", 271.910004},
				{"float-151", 272.910004},
				{"float-152", 273.910004},
				{"float-153", 274.910004},
				{"float-154", 275.910004},
				{"float-155", 276.910004},
				{"float-156", 277.910004},
				{"float-157", 278.910004},
				{"float-158", 279.910004},
			},
			[]*model.DeviceMetaTag{},
		},
		{"dorma1",
			"09C0AEF400E31D01019D040A601CC04980B301A603CC089813302A605CC0D8800B9E04804600",
			[]TestPoint{
				{"char-1", 65.000000},
				{"bool-2", 0.000000},
				{"bool-3", 0.000000},
				{"bool-4", 0.000000},
				{"bool-5", 0.000000},
				{"bool-6", 0.000000},
				{"bool-7", 0.000000},
				{"bool-8", 0.000000},
				{"bool-9", 0.000000},
				{"bool-10", 0.000000},
				{"bool-11", 0.000000},
				{"uint_32-13", 3045394.000000},
			},
			[]*model.DeviceMetaTag{},
		},
	}

	runTests(tests, mockDevice, t)
}

func TestMicroEdgePayload(t *testing.T) {
	test = t
	mockDevice := &model.Device{
		Name: "MicroEdge",
		CommonDevice: model.CommonDevice{
			Model: "MicroEdgeV1",
		},
	}

	tests := []TestStruct{
		{"MicroEdgeOne",
			"17AC7BB100000000FF03FF03FF03FF014B5F",
			[]TestPoint{
				{"pulse", 0.000000},
				{"voltage", 5.100000},
				{"ai_1", 1023.000000},
				{"ai_2", 1023.000000},
				{"ai_3", 1023.000000},
			},
			[]*model.DeviceMetaTag{},
		},
		{"MicroEdgeTwo",
			"55ACA79B00000000FF03FF03FF03FF013F64",
			[]TestPoint{
				{"pulse", 0.000000},
				{"voltage", 5.100000},
				{"ai_1", 1023.000000},
				{"ai_2", 1023.000000},
				{"ai_3", 1023.000000},
			},
			[]*model.DeviceMetaTag{},
		},
	}

	runTests(tests, mockDevice, t)
}

func TestDropletPayload(t *testing.T) {
	test = t
	mockDevice := &model.Device{
		Name: "Droplet",
		CommonDevice: model.CommonDevice{
			Model: "THLM",
		},
	}

	tests := []TestStruct{
		{"DropletOne",
			"CBB272EAB20696263C0000DD000000041861",
			[]TestPoint{
				{"temperature", 17.140000},
				{"pressure", 987.800000},
				{"humidity", 60.000000},
				{"voltage", 4.420000},
				{"light", 0.000000},
				{"motion", 0.000000},
			},
			[]*model.DeviceMetaTag{},
		},
		{"DropletTwo",
			"1AB22D4F2006C6263D0200DB000000E81A62",
			[]TestPoint{
				{"temperature", 15.680000},
				{"pressure", 992.600000},
				{"humidity", 61.000000},
				{"voltage", 4.380000},
				{"light", 2.000000},
				{"motion", 0.000000},
			},
			[]*model.DeviceMetaTag{},
		},
	}

	runTests(tests, mockDevice, t)
}

func TestZHTPayload(t *testing.T) {
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
		{"ZHT-Pub1",
			"00C033AA01D628030101CF0352004E023A01FFFFFFFF251D0000000000000000000000000000000F000100000000004200",
			[]TestPoint{
				{"rebooted", 0.000000},
				{"sleep_mode_status", 1.000000},
				{"temperature_ntc_boiling", 97.500000},
				{"temperature_ntc_chilled", 8.200000},
				{"temperature_ntc_stream", 59.000000},
				{"temperature_ntc_condensor", 31.400000},
				{"fault_1", 255.000000},
				{"fault_2", 255.000000},
				{"fault_3", 255.000000},
				{"fault_4", 255.000000},
				{"usage_energy_kwh", 746.100037},
				{"usage_water_delta_dispenses_boiling", 0.000000},
				{"usage_water_delta_dispenses_chilled", 0.000000},
				{"usage_water_delta_dispenses_sparkling", 0.000000},
				{"usage_water_delta_litres_boiling", 0.000000},
				{"usage_water_delta_litres_chilled", 0.000000},
				{"usage_water_delta_litres_sparkling", 0.000000},
				{"filter_warning_internal", 0.000000},
				{"filter_warning_external", 0.000000},
				{"filter_info_usage_litres_internal", 15.000000},
				{"filter_info_usage_days_internal", 1.000000},
				{"filter_info_usage_litres_external", 0.000000},
				{"filter_info_usage_days_external", 0.000000},
				{"filter_info_usage_litres_uv", 0.000000},
				{"filter_info_usage_days_uv", 0.000000},
				{"filter_warning_uv", 0.000000},
				{"co2_low_gas_warning", 0.000000},
				{"co2_usage_grams", 0.000000},
				{"co2_usage_days", 0.000000},
			},
			[]*model.DeviceMetaTag{},
		},
		{"ZHT-Static",
			"00C032AA010061010102010101323031353036323537353039330000424353203234302F31373500000000000000000035313533415500000000000000000000000000004230332E312E30300000000000000000000000000000FF0708171B0618A922FFFFFFFFFF4E00",
			[]TestPoint{},
			[]*model.DeviceMetaTag{
				{DeviceUUID: "", Key: "firmware_version", Value: "B03.1.00"}, {DeviceUUID: "", Key: "filter_log_date_internal", Value: "27/6/24"}, {DeviceUUID: "", Key: "filter_log_date_external", Value: "255/255/255"}, {DeviceUUID: "", Key: "lora_firmware_major", Value: "2"}, {DeviceUUID: "", Key: "lora_firmware_minor", Value: "1"}, {DeviceUUID: "", Key: "serial_number", Value: "2015062575093"}, {DeviceUUID: "", Key: "model_number", Value: "BCS 240/175"}, {DeviceUUID: "", Key: "product_number", Value: "5153AU"}, {DeviceUUID: "", Key: "modbus_address", Value: "50"}, {DeviceUUID: "", Key: "lora_build_major", Value: "1"}, {DeviceUUID: "", Key: "lora_build_minor", Value: "1"}, {DeviceUUID: "", Key: "calibration_date", Value: "0/0/255"}, {DeviceUUID: "", Key: "filter_log_litres_external", Value: "65535"}, {DeviceUUID: "", Key: "first_50_litres_data", Value: "7/8/23"}, {DeviceUUID: "", Key: "filter_log_litres_internal", Value: "8873"}, {DeviceUUID: "", Key: "filter_log_date_uv", Value: ""}, {DeviceUUID: "", Key: "filter_log_litres_uv", Value: "0"},
			},
		},
		{"ZHT-Write",
			"00C032AA01013302013812C7660F0F0FD40305050070170C00000006D204CC290000CC290000CC290000CC290000CC290000CC290000CC2900004E00",
			[]TestPoint{
				{"time_start_0", 700.000000},
				{"time_stop_0", 0.000000},
				{"enable_start_0", 1.000000},
				{"enable_stop_0", 0.000000},
				{"time_start_1", 700.000000},
				{"time_stop_1", 0.000000},
				{"enable_start_1", 1.000000},
				{"enable_stop_1", 0.000000},
				{"time_start_2", 700.000000},
				{"time_stop_2", 0.000000},
				{"enable_start_2", 1.000000},
				{"enable_stop_2", 0.000000},
				{"time_start_3", 700.000000},
				{"time_stop_3", 0.000000},
				{"enable_start_3", 1.000000},
				{"enable_stop_3", 0.000000},
				{"time_start_4", 700.000000},
				{"time_stop_4", 0.000000},
				{"enable_start_4", 1.000000},
				{"enable_stop_4", 0.000000},
				{"time_start_5", 700.000000},
				{"time_stop_5", 0.000000},
				{"enable_start_5", 1.000000},
				{"enable_stop_5", 0.000000},
				{"time_start_6", 700.000000},
				{"time_stop_6", 0.000000},
				{"enable_start_6", 1.000000},
				{"enable_stop_6", 0.000000},
				{"time", 1724322360.000000},
				{"dispense_time_boiling", 15.000000},
				{"dispense_time_chilled", 15.000000},
				{"dispense_time_sparkling", 15.000000},
				{"temperature_sp_boiling", 98.000000},
				{"temperature_sp_chilled", 5.000000},
				{"temperature_sp_sparkling", 5.000000},
				{"sleep_mode_setting", 0.000000},
				{"filter_info_life_litres_internal", 6000.000000},
				{"filter_info_life_months_internal", 12.000000},
				{"filter_info_life_litres_external", 0.000000},
				{"filter_info_life_months_external", 0.000000},
				{"safety_allow_tap_changes", 1.000000},
				{"safety_lock", 1.000000},
				{"safety_hot_isolation", 0.000000},
				{"security_enable", 0.000000},
				{"security_pin", 1234.000000},
				{"filter_info_life_litres_uv", 0.000000},
				{"filter_info_life_months_uv", 0.000000},
				{"co2_life_grams", 0.000000},
				{"co2_life_months", 0.000000},
				{"co2_pressure", 0.000000},
				{"co2_tank_capacity", 0.000000},
				{"co2_absorption_rate", 0.000000},
				{"sparkling_flow_rate", 0.000000},
				{"sparkling_flush_time", 0.000000},
			},
			[]*model.DeviceMetaTag{},
		},
		{"ZHT-Publish",
			"00C032AA010228030121D4032800C2019801FFFFFFFFA62D00000000000000000000000000000037073700000000004F00",
			[]TestPoint{
				{"rebooted", 1.000000},
				{"sleep_mode_status", 33.000000},
				{"temperature_ntc_boiling", 98.000000},
				{"temperature_ntc_chilled", 4.000000},
				{"temperature_ntc_stream", 45.000000},
				{"temperature_ntc_condensor", 40.799999},
				{"fault_1", 255.000000},
				{"fault_2", 255.000000},
				{"fault_3", 255.000000},
				{"fault_4", 255.000000},
				{"usage_energy_kwh", 1168.599976},
				{"usage_water_delta_dispenses_boiling", 0.000000},
				{"usage_water_delta_dispenses_chilled", 0.000000},
				{"usage_water_delta_dispenses_sparkling", 0.000000},
				{"usage_water_delta_litres_boiling", 0.000000},
				{"usage_water_delta_litres_chilled", 0.000000},
				{"usage_water_delta_litres_sparkling", 0.000000},
				{"filter_warning_internal", 0.000000},
				{"filter_warning_external", 0.000000},
				{"filter_info_usage_litres_internal", 1847.000000},
				{"filter_info_usage_days_internal", 55.000000},
				{"filter_info_usage_litres_external", 0.000000},
				{"filter_info_usage_days_external", 0.000000},
				{"filter_info_usage_litres_uv", 0.000000},
				{"filter_info_usage_days_uv", 0.000000},
				{"filter_warning_uv", 0.000000},
				{"co2_low_gas_warning", 0.000000},
				{"co2_usage_grams", 0.000000},
				{"co2_usage_days", 0.000000},
			},
			[]*model.DeviceMetaTag{},
		},
	}

	runTests(tests, mockDevice, t)
}
