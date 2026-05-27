package pkg

import (
	"strings"
	"testing"

	"github.com/NubeIO/module-core-loraraw/codec"
	"github.com/NubeIO/module-core-loraraw/schema"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
)

// testDefaultKey is declared in payload_test.go and shared across tests.

// Reusable noop callbacks.
var (
	noopPointErr = func(_ string, _ error, _ *model.Device, _ *codec.LoRaDeviceDescription) error {
		return nil
	}
	noopMetaTags   = func(_ string, _ []*model.DeviceMetaTag) error { return nil }
	noopWrittenOK  = func(_ string, _ float64, _ uint8, _ *model.Device) error { return nil }
	noopWrittenErr = func(_ string, _ error, _ uint8, _ *model.Device) error { return nil }
)

// newMockGetDevice returns a getDevice closure that simulates a real DB
// lookup. Bound mocks (mockDevice.AddressUUID set) hit only on a matching
// address — what production sees for a known LoRaRAW device. Unbound mocks
// (AddressUUID nil, e.g. legacy fixtures whose wire address is random
// ciphertext) miss on the wire address and hit on any OTHER address,
// representing "DB knows about the post-decrypt device but not the
// pre-decrypt one." This is purely address-driven; it does NOT depend on
// dispatchFrame's call count, so the test stays correct even if the
// dispatcher's lookup pattern changes.
func newMockGetDevice(mockDevice *model.Device, wireAddr string) func(string) *model.Device {
	bound := mockDevice.AddressUUID != nil
	return func(addr string) *model.Device {
		if bound {
			if strings.EqualFold(*mockDevice.AddressUUID, addr) {
				return mockDevice
			}
			return nil
		}
		// Unbound: pretend the wire address is unknown (legacy ciphertext)
		// but the decrypted address resolves to the mock device.
		if strings.EqualFold(addr, wireAddr) {
			return nil
		}
		return mockDevice
	}
}

// assertPoints verifies that `got` contains every expected (name, value)
// pair within float-equality tolerance. Extra points are logged only when
// `testing -v` is on — they're useful for fixture development but noise in
// CI summaries.
func assertPoints(t *testing.T, want []TestPoint, got map[string]float64) {
	t.Helper()
	expected := make(map[string]float64, len(want))
	for _, p := range want {
		expected[p.Name] = p.Value
	}
	for name, v := range expected {
		gv, ok := got[name]
		if !ok {
			t.Errorf("missing point %q (expected %v)", name, v)
			continue
		}
		if !almostEqual(gv, v) {
			t.Errorf("point %q: expected %v, got %v", name, v, gv)
		}
	}
	if testing.Verbose() {
		for name, v := range got {
			if _, ok := expected[name]; !ok {
				t.Logf("extra point %q = %v (not asserted)", name, v)
			}
		}
	}
}

// runDispatchTests drives every fixture (legacy AES, legacy plaintext,
// encrypted LoRaRAW, plaintext LoRaRAW) through the PRODUCTION dispatcher
// `m.dispatchFrame` (see pkg/app.go). The only test-side wiring is:
//
//   - An address-keyed mock `getDevice` (see newMockGetDevice).
//   - A `capture` callback that stores emitted point values in a map.
//   - Package-level noop callbacks for everything else.
//
// This keeps tests from drifting when handleSerialPayload's branching
// changes — both production and tests call the same dispatchFrame.
func runDispatchTests(tests []TestStruct, mockDevice *model.Device, t *testing.T) {
	t.Helper()
	m := &Module{config: &Config{DefaultKey: testDefaultKey}}
	for _, tt := range tests {
		tt := tt // capture for parallel safety / loop-variable hygiene
		t.Run(tt.Name, func(t *testing.T) {
			wireAddr, err := codec.DecodeAddressHex(tt.Data)
			if err != nil {
				t.Fatalf("decode wire address: %s", err)
			}
			wireAddr = strings.ToUpper(wireAddr)

			got := map[string]float64{}
			capture := func(name string, value float64, _ *model.Device, _ *codec.LoRaDeviceDescription) error {
				got[name] = value
				return nil
			}

			res := m.dispatchFrame(
				tt.Data,
				newMockGetDevice(mockDevice, wireAddr),
				capture, noopPointErr, noopMetaTags,
				noopWrittenOK, noopWrittenErr,
			)
			if !res.OK {
				t.Fatalf("dispatchFrame returned not-OK")
			}
			// Mirror handleSerialPayload: rssi/snr are emitted as ordinary
			// points after dispatchFrame returns. Injecting them here lets
			// fixtures assert them alongside sensor points and keeps the
			// test view of "what got published" identical to production.
			got[codec.RssiField] = float64(res.RSSI)
			got[codec.SnrField] = float64(res.SNR)
			t.Logf("dispatch: address=%s model=%s legacy=%v rssi=%d snr=%.2f publishRawHex=%s",
				res.Address, res.Device.Model, res.LegacyDevice, res.RSSI, res.SNR, res.PublishRawHex)

			assertPoints(t, tt.Values, got)
		})
	}
}

// TestMicroEdgeV1Payload ...
func TestMicroEdgeV1Payload(t *testing.T) {
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

	runDispatchTests(tests, mockDevice, t)
}

// TestEncryptedDropletPayload ...
func TestEncryptedDropletPayload(t *testing.T) {
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
		// Real legacy-encrypted Droplet frame captured from production logs on
		// 2026-05-14. Wire address 355055BD decrypts (default key) to device
		// address C3B2B971 -> THLM model "dr3". Regression for the auto-detect
		// legacy fallback path in handleSerialPayload.
		//
		// Also locks in the rssi/snr fix: the original wire bytes end in
		// 0x44, 0x27. tryLegacyDecrypt must re-append these to the decrypted
		// dataHex so DecodeRSSI/DecodeSNR see the real radio metadata
		// (rssi=-68, snr=9.75) instead of the last 2 plaintext sensor bytes.
		{"DropletLegacyAutoDetect",
			"355055BD6FE9265D44A4B9B1D20BCA234427",
			[]TestPoint{
				{"temperature", 21.830000},
				{"pressure", 1023.700000},
				{"humidity", 56.000000},
				{"voltage", 4.480000},
				{"light", 2.000000},
				{"motion", 0.000000},
				{"rssi", -68},
				{"snr", 9.75},
			},
			[]*model.DeviceMetaTag{},
		},
	}

	runDispatchTests(tests, mockDevice, t)
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
		// Real encrypted ZHT frame captured from production logs on 2026-05-14.
		// Verifies the auto-detect "frame longer than plaintext" path:
		// handleSerialPayload should pick the CMAC-verified decrypt branch and
		// produce the same point values the live pipeline published over MQTT.
		{
			Name: "ZHT-Encrypted-AutoDetect",
			Data: "00C032AA34B3C68E97813072CE3987B7BEB8C49B21054F6CF1DEAADDBF086DCE5560AA400442F4616745251129C3920C4002C373FF83C3E6F2BE10A4B8C37E87193A374F369E283F5627",
			Values: []TestPoint{
				{"rebooted", 0}, {"sleep_mode_status", 1},
				{"temperature_ntc_boiling", 97.5}, {"temperature_ntc_chilled", 8.2},
				{"temperature_ntc_stream", 61.5}, {"temperature_ntc_condensor", 30.4},
				{"fault_1", 255}, {"fault_2", 255}, {"fault_3", 255}, {"fault_4", 255},
				{"usage_energy_kwh", 4625.5},
				{"usage_water_delta_dispenses_boiling", 0}, {"usage_water_delta_dispenses_chilled", 0}, {"usage_water_delta_dispenses_sparkling", 0},
				{"usage_water_delta_litres_boiling", 0}, {"usage_water_delta_litres_chilled", 0}, {"usage_water_delta_litres_sparkling", 0},
				{"filter_warning_internal", 0}, {"filter_warning_external", 0},
				{"filter_info_usage_litres_internal", 793}, {"filter_info_usage_days_internal", 181},
				{"filter_info_usage_litres_external", 0}, {"filter_info_usage_days_external", 0},
				{"filter_info_usage_litres_uv", 0}, {"filter_info_usage_days_uv", 0},
				{"filter_warning_uv", 0}, {"co2_low_gas_warning", 0},
				{"co2_usage_grams", 0}, {"co2_usage_days", 0},
			},
			MetaTags: []*model.DeviceMetaTag{},
		},
	}

	runDispatchTests(tests, mockDevice, t)
}

// TestEncryptedRubixPayload ...
func TestEncryptedRubixPayload(t *testing.T) {
	test = t
	addr := "5CC08E7B"
	mockDevice := &model.Device{
		Name: "Rubix",
		CommonDevice: model.CommonDevice{
			Model:       "Rubix",
			AddressUUID: &addr,
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

	runDispatchTests(tests, mockDevice, t)
}

// TestEncryptedUARTPayload replays real encrypted LoRaRAW frames captured
// from a UART device (address C3C0A660, dev_333842e4715b479e, name "AC3")
// on 2026-05-14. Verifies the auto-detect encrypted path end-to-end:
//
//	handleSerialPayload sees devDesc.IsLoRaRAW == true, isUnencryptedLoRaRAW
//	is false (frame is longer than the plaintext layout), so it calls
//	decryptLoRaRAWPkt + buildUnencryptedRawFrame and forwards to the codec.
//
// Expected values are taken directly from the MQTT `module-core-loraraw/value`
// publish in the production logs.
func TestEncryptedUARTPayload(t *testing.T) {
	test = t
	addr := "C3C0A660"
	mockDevice := &model.Device{
		Name: "AC3",
		CommonDevice: model.CommonDevice{
			Model:       schema.DeviceModelUART,
			AddressUUID: &addr,
		},
	}

	tests := []TestStruct{
		// Full 30-point uplink (decodedLen=88 in the logs). The two captures
		// differ only in nonce/CMAC; both decode to the same point values.
		{
			Name: "UART-Encrypted-Full-1",
			Data: "C3C0A660F96AD89881DA2DC39D83CFDEFA2174E8B5FBB30D6F920AD8943E394DFF5E79763D2F42D61BEF3A4BDAB28BA3241D6A2A61CB7ED85F32788A2C619C23A9CC4A609E93DB8428BB38AC1C46F8845438F7555D1022023226",
			Values: []TestPoint{
				{"UVP-1", 1}, {"UVP-2", 0}, {"UVP-3", 1},
				{"UVP-10", 1}, {"UVP-11", 1}, {"UVP-12", 1}, {"UVP-13", 1},
				{"UVP-14", 1}, {"UVP-15", 1}, {"UVP-16", 1}, {"UVP-17", 1},
				{"UVP-18", 1}, {"UVP-19", 1}, {"UVP-20", 4}, {"UVP-21", 1},
				{"UVP-30", 5}, {"UVP-31", 1},
				{"UVP-40", 1}, {"UVP-41", 1}, {"UVP-42", 26}, {"UVP-43", 8},
				{"UVP-44", 1}, {"UVP-45", 1},
				{"UVP-54", 3}, {"UVP-55", 0}, {"UVP-56", 26},
				{"UVP-57", 0}, {"UVP-59", 0},
			},
			MetaTags: []*model.DeviceMetaTag{},
		},
		{
			Name: "UART-Encrypted-Full-2",
			Data: "C3C0A660C0EEEBFDFC8D2AB57D42481080684F6F8CEB758C842B0A083DA9FFDE4137A6D20AE0CF061583EB9601690DF104BB10B9CD0114DA6A670391D89FCA329D97AF0E0ED6DD46F7178EE679DE31E19D8417023BFED4B83127",
			Values: []TestPoint{
				{"UVP-1", 1}, {"UVP-2", 0}, {"UVP-3", 1},
				{"UVP-10", 1}, {"UVP-11", 1}, {"UVP-12", 1}, {"UVP-13", 1},
				{"UVP-14", 1}, {"UVP-15", 1}, {"UVP-16", 1}, {"UVP-17", 1},
				{"UVP-18", 1}, {"UVP-19", 1}, {"UVP-20", 4}, {"UVP-21", 1},
				{"UVP-30", 5}, {"UVP-31", 1},
				{"UVP-40", 1}, {"UVP-41", 1}, {"UVP-42", 26}, {"UVP-43", 8},
				{"UVP-44", 1}, {"UVP-45", 1},
				{"UVP-54", 3}, {"UVP-55", 0}, {"UVP-56", 26},
				{"UVP-57", 0}, {"UVP-59", 0},
			},
			MetaTags: []*model.DeviceMetaTag{},
		},
		// Short response uplink (decodedLen=24 in the logs). The codec emits
		// no UVP points for this frame; the live pipeline only publishes
		// rssi/snr. Asserting an empty Values map proves the decryption +
		// dispatch path doesn't error out and doesn't fabricate points.
		{
			Name:     "UART-Encrypted-Response",
			Data:     "C3C0A66062CAF12AF46A435170DDE74446D38293C4CE47B33226",
			Values:   []TestPoint{},
			MetaTags: []*model.DeviceMetaTag{},
		},
	}

	runDispatchTests(tests, mockDevice, t)
}

// ─────────────────────────────────────────────────────────────────────────────
// Unencrypted (plaintext) fixtures follow.
// ─────────────────────────────────────────────────────────────────────────────

// TestUnencryptedTHLPayload replays a real PLAINTEXT (unencrypted) LEGACY
// frame captured on 2026-05-15 from a THL device "dr3" at address 3DB20FF3
// (dev_dcf24b293849476d). The dispatcher should:
//
//  1. Find the device on the initial lookup (address matches the mock).
//  2. Skip tryLegacyDecrypt entirely (initial lookup hit).
//  3. See devDesc.IsLoRaRAW == false and take the legacy plaintext handler
//     path — no decryption, no LoRaRAW header stripping; the codec consumes
//     the full wire frame.
//
// THL is a stripped-down Droplet variant: it emits temperature/pressure/
// humidity/voltage/light but NOT motion (5 sensor points instead of 6).
// Expected values are taken verbatim from the live MQTT publish:
//
//	topic=module-core-loraraw/value device=dr3 points=7
//	(5 sensor points + rssi + snr; only the 5 sensor points are asserted)
func TestUnencryptedTHLPayload(t *testing.T) {
	test = t

	// dr3 @ 3DB20FF3 (dev_dcf24b293849476d)
	addrDr3 := "3DB20FF3"
	mockDr3 := &model.Device{
		Name: "dr3",
		CommonDevice: model.CommonDevice{
			Model:       "THL",
			AddressUUID: &addrDr3,
		},
	}
	runDispatchTests([]TestStruct{
		{
			Name: "THL-Plaintext-dr3",
			Data: "3DB20FF3400868262B0000D9D101B3E43400",
			Values: []TestPoint{
				{"temperature", 21.12},
				{"pressure", 983.2},
				{"humidity", 43},
				{"voltage", 4.34},
				{"light", 0},
			},
			MetaTags: []*model.DeviceMetaTag{},
		},
	}, mockDr3, t)

	// Second real plaintext-legacy THL frame captured on 2026-05-15 from a
	// DIFFERENT physical device "dr2" at address 10B2698C
	// (dev_adf0902cf7684154). Same dispatch path as dr3 above; different
	// address proves the codec doesn't accidentally hard-code anything to a
	// specific device. Bound to its own mock so the initial DB lookup hits
	// (otherwise the dispatcher would mistakenly try legacy auto-decrypt).
	addrDr2 := "10B2698C"
	mockDr2 := &model.Device{
		Name: "dr2",
		CommonDevice: model.CommonDevice{
			Model:       "THL",
			AddressUUID: &addrDr2,
		},
	}
	runDispatchTests([]TestStruct{
		{
			Name: "THL-Plaintext-dr2",
			Data: "10B2698CB7076C262E0000D987B014692C00",
			Values: []TestPoint{
				{"temperature", 19.75},
				{"pressure", 983.6},
				{"humidity", 46},
				{"voltage", 4.34},
				{"light", 0},
			},
			MetaTags: []*model.DeviceMetaTag{},
		},
	}, mockDr2, t)
}

func TestUnencryptedMicroEdgeV2Payload(t *testing.T) {
	test = t

	// dr3 @ 3DB20FF3 (dev_dcf24b293849476d)
	addrDr3 := "69ACF3A2"
	mockDr3 := &model.Device{
		Name: "MicroEdgeV2",
		CommonDevice: model.CommonDevice{
			Model:       "MicroEdgeV2",
			AddressUUID: &addrDr3,
		},
	}
	runDispatchTests([]TestStruct{
		{
			Name: "MicroEdgeV2-WithTrailer",
			// Wire layout (18 bytes):
			//   [addr:4][pulse:4][voltage:1][ai1:2][ai2:2][ai3:2][flags:1][rssi:1][snr:1]
			// Trailing 2 bytes (5B 27) are appended by the SX127x driver
			// pipeline (_serial_lora_format):
			//   rssi byte 0x5B = 91   -> DecodeRSSI: -91
			//   snr  byte 0x27 = 39   -> DecodeSNR : 39/4 = 9.75
			Data: "69ACF3A20002EA9DB803FF03FF03FF015B27",
			Values: []TestPoint{
				{"pulse", 191133},
				{"voltage", 3.68},
				{"ai_1", 1023},
				{"ai_2", 1023},
				{"ai_3", 1023},
				{"rssi", -91},
				{"snr", 9.75},
			},
			MetaTags: []*model.DeviceMetaTag{},
		},
	}, mockDr3, t)
}

// TestUnencryptedZHTPayload replays a real PLAINTEXT (unencrypted) LoRaRAW
// frame captured on 2026-05-15 from a ZipHydroTap with address 00C03200
// (device zht-1, dev_4d0cd6dfec1b4485). The dispatcher should:
//
//  1. Find the device on the initial lookup (address matches the mock).
//  2. Detect the frame as unencrypted LoRaRAW because its length matches the
//     plaintext layout exactly (no AES padding, no CMAC).
//  3. Strip the LoRaRAW header (addr+opts+nonce+len) and decode the inner
//     payload through DecodeZHT.
//
// Expected values are taken verbatim from the live MQTT publish:
//
//	topic=module-core-loraraw/value device=zht-1 points=31
func TestUnencryptedZHTPayload(t *testing.T) {
	test = t
	addr := "00C03200"
	mockDevice := &model.Device{
		Name: "zht-1",
		CommonDevice: model.CommonDevice{
			Model:       schema.DeviceModelZiptHydroTap,
			AddressUUID: &addr,
		},
	}

	// Two consecutive captures (sec 17 and sec 02) that differ only in the
	// 1-byte nonce field (0x4C vs 0x4D). Both decode to the same point
	// values — useful as a nonce-independence regression.
	tests := []TestStruct{
		{
			Name: "ZHT-Plaintext-1",
			Data: "00C03200014C28030101C5034E000D021A01FFFFFFFFD53500000000000000000000000000000002012C00000000004100",
			Values: []TestPoint{
				{"rebooted", 0}, {"sleep_mode_status", 1},
				{"temperature_ntc_boiling", 96.5}, {"temperature_ntc_chilled", 7.800000190734863},
				{"temperature_ntc_stream", 52.5}, {"temperature_ntc_condensor", 28.200000762939453},
				{"fault_1", 255}, {"fault_2", 255}, {"fault_3", 255}, {"fault_4", 255},
				{"usage_energy_kwh", 1378.0999755859375},
				{"usage_water_delta_dispenses_boiling", 0}, {"usage_water_delta_dispenses_chilled", 0}, {"usage_water_delta_dispenses_sparkling", 0},
				{"usage_water_delta_litres_boiling", 0}, {"usage_water_delta_litres_chilled", 0}, {"usage_water_delta_litres_sparkling", 0},
				{"filter_warning_internal", 0}, {"filter_warning_external", 0},
				{"filter_info_usage_litres_internal", 258}, {"filter_info_usage_days_internal", 44},
				{"filter_info_usage_litres_external", 0}, {"filter_info_usage_days_external", 0},
				{"filter_info_usage_litres_uv", 0}, {"filter_info_usage_days_uv", 0},
				{"filter_warning_uv", 0}, {"co2_low_gas_warning", 0},
				{"co2_usage_grams", 0}, {"co2_usage_days", 0},
			},
			MetaTags: []*model.DeviceMetaTag{},
		},
		{
			Name: "ZHT-Plaintext-2",
			Data: "00C03200014D28030101C5034E000D021A01FFFFFFFFD53500000000000000000000000000000002012C00000000004100",
			Values: []TestPoint{
				{"rebooted", 0}, {"sleep_mode_status", 1},
				{"temperature_ntc_boiling", 96.5}, {"temperature_ntc_chilled", 7.800000190734863},
				{"temperature_ntc_stream", 52.5}, {"temperature_ntc_condensor", 28.200000762939453},
				{"fault_1", 255}, {"fault_2", 255}, {"fault_3", 255}, {"fault_4", 255},
				{"usage_energy_kwh", 1378.0999755859375},
				{"usage_water_delta_dispenses_boiling", 0}, {"usage_water_delta_dispenses_chilled", 0}, {"usage_water_delta_dispenses_sparkling", 0},
				{"usage_water_delta_litres_boiling", 0}, {"usage_water_delta_litres_chilled", 0}, {"usage_water_delta_litres_sparkling", 0},
				{"filter_warning_internal", 0}, {"filter_warning_external", 0},
				{"filter_info_usage_litres_internal", 258}, {"filter_info_usage_days_internal", 44},
				{"filter_info_usage_litres_external", 0}, {"filter_info_usage_days_external", 0},
				{"filter_info_usage_litres_uv", 0}, {"filter_info_usage_days_uv", 0},
				{"filter_warning_uv", 0}, {"co2_low_gas_warning", 0},
				{"co2_usage_grams", 0}, {"co2_usage_days", 0},
			},
			MetaTags: []*model.DeviceMetaTag{},
		},
	}

	runDispatchTests(tests, mockDevice, t)
}

// TestUnencryptedRubixPayload replays a real PLAINTEXT (unencrypted) LoRaRAW
// frame captured on 2026-05-15 from a Rubix device "Dorma-1" with address
// ACC0D08A (dev_36140a4d1e774982). Same plaintext-LoRaRAW path as
// TestUnencryptedZHTPayload but with the Rubix codec, which decodes the
// inner payload as a tag-length-value (TLV) stream and emits a mix of
// bool / char / uint_32 typed points.
//
// Expected values are taken verbatim from the live MQTT publish:
//
//	topic=module-core-loraraw/value device=Dorma-1 points=14
func TestUnencryptedRubixPayload(t *testing.T) {
	test = t
	addr := "ACC0D08A"
	mockDevice := &model.Device{
		Name: "Dorma-1",
		CommonDevice: model.CommonDevice{
			Model:       "Rubix",
			AddressUUID: &addr,
		},
	}

	tests := []TestStruct{
		{
			Name: "Rubix-Plaintext-Dorma-1",
			Data: "ACC0D08A00371D01019D300A601CC04980B301A603CC089813302A685CC0D88000C0F0003000",
			Values: []TestPoint{
				{"char-2", 76},
				{"bool-3", 0}, {"bool-4", 0}, {"bool-5", 0}, {"bool-6", 0},
				{"bool-7", 0}, {"bool-8", 0}, {"bool-9", 0}, {"bool-10", 0},
				{"bool-11", 1}, {"bool-12", 0},
				{"uint_32-14", 197568},
			},
			MetaTags: []*model.DeviceMetaTag{},
		},
	}

	runDispatchTests(tests, mockDevice, t)
}
