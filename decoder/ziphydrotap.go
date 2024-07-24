package decoder

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/NubeIO/module-core-loraraw/utils"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
	"strconv"
)

type TZHTPayloadType int

const (
	ZHTPlLenStaticV1 = 97
	ZHTPlLenStaticV2 = 102 // 9500ms
	ZHTPlLenWriteV1  = 51
	ZHTPlLenWriteV2  = 66 // 7200ms
	ZHTPlLenPollV1   = 40
	ZHTPlLenPollV2   = 47 // 6200ms
	ZipHTTimerLength = 7
)

const (
	RebootField            = "reboot"
	ResetFilterField       = "reset_filter"
	RemoteCalibrationField = "remote_calibration"
	ResetEnergyField       = "reset_energy"
)

const (
	TimeStartField   = "time_start"
	TimeStopField    = "time_stop"
	EnableStartField = "enable_start"
	EnableStopField  = "enable_stop"
)

const (
	TimeField                         = "time"
	DispenseTimeBoilingField          = "dispense_time_boiling"
	DispenseTimeChilledField          = "dispense_time_chilled"
	DispenseTimeSparklingField        = "dispense_time_sparkling"
	TemperatureSPBoilingField         = "temperature_sp_boiling"
	TemperatureSPChilledField         = "temperature_sp_chilled"
	TemperatureSPSparklingField       = "temperature_sp_sparkling"
	SleepModeSettingField             = "sleep_mode_setting"
	FilterInfoLifeLitresInternalField = "filter_info_life_litres_internal"
	FilterInfoLifeMonthsInternalField = "filter_info_life_months_internal"
	FilterInfoLifeLitresExternalField = "filter_info_life_litres_external"
	FilterInfoLifeMonthsExternalField = "filter_info_life_months_external"
	SafetyAllowTapChangesField        = "safety_allow_tap_changes"
	SafetyLockField                   = "safety_lock"
	SafetyHotIsolationField           = "safety_hot_isolation"
	SecurityEnableField               = "security_enable"
	SecurityPinField                  = "security_pin"
	FilterInfoLifeLitresUVField       = "filter_info_life_litres_uv"
	FilterInfoLifeMonthsUVField       = "filter_info_life_months_uv"
	CO2LifeGramsField                 = "co2_life_grams"
	CO2LifeMonthsField                = "co2_life_months"
	CO2PressureField                  = "co2_pressure"
	CO2TankCapacityField              = "co2_tank_capacity"
	CO2AbsorptionRateField            = "co2_absorption_rate"
	SparklingFlowRateField            = "sparkling_flow_rate"
	SparklingFlushTimeField           = "sparkling_flush_time"
	// TimersField                       = "timers" // Is added through loop
)

const (
	RebootedField                          = "rebooted"
	SleepModeStatusField                   = "sleep_mode_status"
	TemperatureNTCBoilingField             = "temperature_ntc_boiling"
	TemperatureNTCChilledField             = "temperature_ntc_chilled"
	TemperatureNTCStreamField              = "temperature_ntc_stream"
	TemperatureNTCCondensorField           = "temperature_ntc_condensor"
	Fault1Field                            = "fault_1"
	Fault2Field                            = "fault_2"
	Fault3Field                            = "fault_3"
	Fault4Field                            = "fault_4"
	UsageEnergyKWhField                    = "usage_energy_kwh"
	UsageWaterDeltaDispensesBoilingField   = "usage_water_delta_dispenses_boiling"
	UsageWaterDeltaDispensesChilledField   = "usage_water_delta_dispenses_chilled"
	UsageWaterDeltaDispensesSparklingField = "usage_water_delta_dispenses_sparkling"
	UsageWaterDeltaLitresBoilingField      = "usage_water_delta_litres_boiling"
	UsageWaterDeltaLitresChilledField      = "usage_water_delta_litres_chilled"
	UsageWaterDeltaLitresSparklingField    = "usage_water_delta_litres_sparkling"
	FilterWarningInternalField             = "filter_warning_internal"
	FilterWarningExternalField             = "filter_warning_external"
	FilterInfoUsageLitresInternalField     = "filter_info_usage_litres_internal"
	FilterInfoUsageDaysInternalField       = "filter_info_usage_days_internal"
	FilterInfoUsageLitresExternalField     = "filter_info_usage_litres_external"
	FilterInfoUsageDaysExternalField       = "filter_info_usage_days_external"
	FilterInfoUsageLitresUVField           = "filter_info_usage_litres_uv"
	FilterInfoUsageDaysUVField             = "filter_info_usage_days_uv"
	FilterWarningUVField                   = "filter_warning_uv"
	CO2LowGasWarningField                  = "co2_low_gas_warning"
	CO2UsageGramsField                     = "co2_usage_grams"
	CO2UsageDaysField                      = "co2_usage_days"
)

const (
	ErrorData = iota
	StaticData
	WriteData
	PollData
)

func DecodeZHT(data string, devDesc *LoRaDeviceDescription, device *model.Device) error {
	updateDeviceFault(devDesc.Model, device.UUID)

	bytes, err := getPayloadBytes(data)
	if err != nil {
		return err
	}

	switch pl := getPayloadType(data); pl {
	// TODO: This should be meta data when it gets supported
	// case StaticData:
	//     payload := staticPayloadDecoder(bytes)
	//     payloadFull := TZipHydrotapStaticFull{TZipHydrotapStatic: payload}
	//     return &payloadFull.CommonValues, payloadFull
	case WriteData:
		err := writePayloadDecoder(bytes, device)
		return err
	case PollData:
		err := pollPayloadDecoder(bytes, device)
		return err
	}

	return nil
}

func getPayloadType(data string) TZHTPayloadType {
	plID, _ := strconv.ParseInt(data[:2], 16, 0)
	return TZHTPayloadType(plID)
}

func CheckPayloadLengthZHT(data string) bool {
	// 4 bytes address | 1 byte opts | 1 byte nonce | 1 byte length | 4 byte cmac | 1 byte rssi | 1 byte snr
	dataLen := len(data)
	payloadLength := dataLen / 2
	payloadLength -= 13

	onlyData := data[14 : dataLen-12]
	payloadType := getPayloadType(onlyData)
	dataLength, _ := strconv.ParseInt(data[12:14], 16, 0)

	if getPacketVersion(data) == 1 {
		return (payloadType == StaticData && dataLength == ZHTPlLenStaticV1 && payloadLength > ZHTPlLenStaticV1) ||
			(payloadType == WriteData && dataLength == ZHTPlLenWriteV1 && payloadLength > ZHTPlLenWriteV1) ||
			(payloadType == PollData && dataLength == ZHTPlLenPollV1 && payloadLength > ZHTPlLenPollV1)
	} else if getPacketVersion(data) == 2 {
		return (payloadType == StaticData && dataLength == ZHTPlLenStaticV2 && payloadLength > ZHTPlLenStaticV2) ||
			(payloadType == WriteData && dataLength == ZHTPlLenWriteV2 && payloadLength > ZHTPlLenWriteV2) ||
			(payloadType == PollData && dataLength == ZHTPlLenPollV2 && payloadLength > ZHTPlLenPollV2)
	}
	return false
}

func GenerateTimerFieldNames() []string {
	var timerFields []string

	for i := 0; i < ZipHTTimerLength; i++ {
		timerFields = append(timerFields,
			fmt.Sprintf("%s_%d", TimeStartField, i),
			fmt.Sprintf("%s_%d", TimeStopField, i),
			fmt.Sprintf("%s_%d", EnableStartField, i),
			fmt.Sprintf("%s_%d", EnableStopField, i),
		)
	}

	return timerFields
}

func GetTZipHydroTapWriteOnlyFields() []string {
	return []string{
		RebootField,
		ResetFilterField,
		RemoteCalibrationField,
		ResetEnergyField,
	}
}

func GetTZipHydroTapWriteFields() []string {
	tZipHydroTapWriteFields := []string{
		TimeField,
		DispenseTimeBoilingField,
		DispenseTimeChilledField,
		DispenseTimeSparklingField,
		TemperatureSPBoilingField,
		TemperatureSPChilledField,
		TemperatureSPSparklingField,
		SleepModeSettingField,
		FilterInfoLifeLitresInternalField,
		FilterInfoLifeMonthsInternalField,
		FilterInfoLifeLitresExternalField,
		FilterInfoLifeMonthsExternalField,
		SafetyAllowTapChangesField,
		SafetyLockField,
		SafetyHotIsolationField,
		SecurityEnableField,
		SecurityPinField,
		FilterInfoLifeLitresUVField,
		FilterInfoLifeMonthsUVField,
		CO2LifeGramsField,
		CO2LifeMonthsField,
		CO2PressureField,
		CO2TankCapacityField,
		CO2AbsorptionRateField,
		SparklingFlowRateField,
		SparklingFlushTimeField,
	}
	return append(tZipHydroTapWriteFields, GenerateTimerFieldNames()...)
}

func GetTZipHydroTapPollFields() []string {
	return []string{
		RebootedField,
		SleepModeStatusField,
		TemperatureNTCBoilingField,
		TemperatureNTCChilledField,
		TemperatureNTCStreamField,
		TemperatureNTCCondensorField,
		UsageEnergyKWhField,
		UsageWaterDeltaDispensesBoilingField,
		UsageWaterDeltaDispensesChilledField,
		UsageWaterDeltaDispensesSparklingField,
		UsageWaterDeltaLitresBoilingField,
		UsageWaterDeltaLitresChilledField,
		UsageWaterDeltaLitresSparklingField,
		Fault1Field,
		Fault2Field,
		Fault3Field,
		Fault4Field,
		FilterWarningInternalField,
		FilterWarningExternalField,
		FilterInfoUsageLitresInternalField,
		FilterInfoUsageDaysInternalField,
		FilterInfoUsageLitresExternalField,
		FilterInfoUsageDaysExternalField,
		FilterInfoUsageLitresUVField,
		FilterInfoUsageDaysUVField,
		FilterWarningUVField,
		CO2LowGasWarningField,
		CO2UsageGramsField,
		CO2UsageDaysField,
	}
}

func GetZHTPointNames() []string {
	commonValueFields := GetCommonValueNames()
	tZipHydroTapWriteOnlyFields := GetTZipHydroTapWriteOnlyFields()
	tZipHydroTapWriteFields := GetTZipHydroTapWriteFields()
	tZipHydroTapPollFields := GetTZipHydroTapPollFields()

	return append(
		append(
			append(
				commonValueFields,
				tZipHydroTapWriteOnlyFields...,
			),
			tZipHydroTapWriteFields...,
		),
		tZipHydroTapPollFields...,
	)
}

func getPayloadBytes(data string) ([]byte, error) {
	bytes, err := hex.DecodeString(data[2:])
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

func getPacketVersion(data string) uint8 {
	v, _ := strconv.ParseInt(data[16:18], 16, 0)
	return uint8(v)
}

func bytesToString(bytes []byte) string {
	str := ""
	for _, b := range bytes {
		if b == 0 {
			break
		}
		str += string(b)
	}
	return str
}

func bytesToDate(bytes []byte) string {
	return fmt.Sprintf("%d/%d/%d", bytes[0], bytes[1], bytes[2])
}

// No usages of staticPayloadDecoder method
// func staticPayloadDecoder(data []byte, device *model.Device) error {
// 	index := 1
// 	fwMa := data[index]
// 	index += 1
// 	fwMi := data[index]
// 	index += 1
// 	buildMa := data[index]
// 	index += 1
// 	buildMi := data[index]
// 	index += 1
// 	sn := bytesToString(data[index : index+15])
// 	index += 15
// 	mn := bytesToString(data[index : index+20])
// 	index += 20
// 	pn := bytesToString(data[index : index+20])
// 	index += 20
// 	fw := bytesToString(data[index : index+20])
// 	index += 20
// 	calDate := bytesToDate(data[index : index+3])
// 	index += 3
// 	f50lDate := bytesToDate(data[index : index+3])
// 	index += 3
// 	filtLogDateInt := bytesToDate(data[index : index+3])
// 	index += 3
// 	filtLogLitresInt := int(binary.LittleEndian.Uint16(data[index : index+2]))
// 	index += 2
// 	filtLogDateExt := bytesToDate(data[index : index+3])
// 	index += 3
// 	filtLogLitresExt := int(binary.LittleEndian.Uint16(data[index : index+2]))
// 	index += 2
//
// 	filtLogDateUV := ""
// 	filtLogLitresUV := 0
// 	if data[0] >= 2 {
// 		filtLogDateUV = bytesToDate(data[index : index+3])
// 		index += 3
// 		filtLogLitresUV = int(binary.LittleEndian.Uint16(data[index : index+2]))
// 		index += 2
// 	}
//
// 	_ = UpdateDevicePoint("lora_firmware_major", float64(fwMa), device)
// 	_ = UpdateDevicePoint("lora_firmware_minor", float64(fwMi), device)
// 	_ = UpdateDevicePoint("lora_build_major", float64(buildMa), device)
// 	_ = UpdateDevicePoint("lora_build_minor", float64(buildMi), device)
// 	_ = UpdateDevicePoint("serial_number", sn, device)
// 	_ = UpdateDevicePoint("model_number", mn, device)
// 	_ = UpdateDevicePoint("product_number", pn, device)
// 	_ = UpdateDevicePoint("firmware_version", fw, device)
// 	_ = UpdateDevicePoint("calibration_date", calDate, device)
// 	_ = UpdateDevicePoint("first_50_litres_data", f50lDate, device)
// 	_ = UpdateDevicePoint("filter_log_date_internal", filtLogDateInt, device)
// 	_ = UpdateDevicePoint("filter_log_litres_internal", filtLogLitresInt, device)
// 	_ = UpdateDevicePoint("filter_log_date_external", filtLogDateExt, device)
// 	_ = UpdateDevicePoint("filter_log_litres_external", filtLogLitresExt, device)
// 	_ = UpdateDevicePoint("filter_log_date_uv", filtLogDateUV, device)
// 	_ = UpdateDevicePoint("filter_log_litres_uv", filtLogLitresUV, device)
//
// 	return nil
// }

func writePayloadDecoder(data []byte, device *model.Device) error {
	index := 1
	time := int(binary.LittleEndian.Uint32(data[index : index+4]))
	index += 4
	dispB := int(data[index])
	index += 1
	dispC := int(data[index])
	index += 1
	dispS := int(data[index])
	index += 1
	tempSpB := float32(binary.LittleEndian.Uint16(data[index:index+2])) / 10
	index += 2
	tempSpC := float32(int(data[index]))
	index += 1
	tempSpS := float32(int(data[index]))
	index += 1
	sm := int(data[index])
	index += 1
	filLyfLtrInt := int(binary.LittleEndian.Uint16(data[index : index+2]))
	index += 2
	filLyfMnthInt := int(data[index])
	index += 1
	filLyfLtrExt := int(binary.LittleEndian.Uint16(data[index : index+2]))
	index += 2
	filLyfMnthExt := int(data[index])
	index += 1
	sfTap := (data[index]>>2)&1 == 1
	sfL := (data[index]>>1)&1 == 1
	sfHi := (data[index]>>0)&1 == 1
	index += 1
	secUI16 := binary.LittleEndian.Uint16(data[index : index+2])
	secEn := secUI16 >= 10000
	secPin := int(secUI16 % 10000)
	index += 2

	var u16 uint16
	for i := 0; i < ZipHTTimerLength; i++ {
		u16 = binary.LittleEndian.Uint16(data[index : index+2])
		timeStart := int(u16 % 10000)
		enableStart := u16 >= 10000
		index += 2
		u16 = binary.LittleEndian.Uint16(data[index : index+2])
		timeStop := int(u16 % 10000)
		enableStop := u16 >= 10000
		index += 2
		_ = UpdateDevicePoint(fmt.Sprintf("%s_%d", TimeStartField, i), float64(timeStart), device)
		_ = UpdateDevicePoint(fmt.Sprintf("%s_%d", TimeStopField, i), float64(timeStop), device)
		_ = UpdateDevicePoint(fmt.Sprintf("%s_%d", EnableStartField, i), utils.BoolToFloat(enableStart), device)
		_ = UpdateDevicePoint(fmt.Sprintf("%s_%d", EnableStopField, i), utils.BoolToFloat(enableStop), device)
	}

	filLyfLtrUV := 0
	filLyfMnthUV := 0
	cO2LyfGrams := 0
	cO2LyfMnths := 0
	cO2Pressure := 0
	cO2TankCap := 0
	cO2AbsorpRate := 0
	sparklFlowRate := 0
	sparklFlushTime := 0
	if data[0] >= 2 {
		filLyfLtrUV = int(binary.LittleEndian.Uint16(data[index : index+2]))
		index += 2
		filLyfMnthUV = int(data[index])
		index += 1
		cO2LyfGrams = int(binary.LittleEndian.Uint16(data[index : index+2]))
		index += 2
		cO2LyfMnths = int(data[index])
		index += 1
		cO2Pressure = int(data[index])
		index += 1
		cO2TankCap = int(binary.LittleEndian.Uint16(data[index : index+2]))
		index += 2
		cO2AbsorpRate = int(binary.LittleEndian.Uint16(data[index : index+2]))
		index += 2
		sparklFlowRate = int(binary.LittleEndian.Uint16(data[index : index+2]))
		index += 2
		sparklFlushTime = int(binary.LittleEndian.Uint16(data[index : index+2]))
		index += 2
	}

	_ = UpdateDevicePoint(TimeField, float64(time), device)
	_ = UpdateDevicePoint(DispenseTimeBoilingField, float64(dispB), device)
	_ = UpdateDevicePoint(DispenseTimeChilledField, float64(dispC), device)
	_ = UpdateDevicePoint(DispenseTimeSparklingField, float64(dispS), device)
	_ = UpdateDevicePoint(TemperatureSPBoilingField, float64(tempSpB), device)
	_ = UpdateDevicePoint(TemperatureSPChilledField, float64(tempSpC), device)
	_ = UpdateDevicePoint(TemperatureSPSparklingField, float64(tempSpS), device)
	_ = UpdateDevicePoint(SleepModeSettingField, float64(sm), device)
	_ = UpdateDevicePoint(FilterInfoLifeLitresInternalField, float64(filLyfLtrInt), device)
	_ = UpdateDevicePoint(FilterInfoLifeMonthsInternalField, float64(filLyfMnthInt), device)
	_ = UpdateDevicePoint(FilterInfoLifeLitresExternalField, float64(filLyfLtrExt), device)
	_ = UpdateDevicePoint(FilterInfoLifeMonthsExternalField, float64(filLyfMnthExt), device)
	_ = UpdateDevicePoint(SafetyAllowTapChangesField, utils.BoolToFloat(sfTap), device)
	_ = UpdateDevicePoint(SafetyLockField, utils.BoolToFloat(sfL), device)
	_ = UpdateDevicePoint(SafetyHotIsolationField, utils.BoolToFloat(sfHi), device)
	_ = UpdateDevicePoint(SecurityEnableField, utils.BoolToFloat(secEn), device)
	_ = UpdateDevicePoint(SecurityPinField, float64(secPin), device)
	// Pkt V2
	_ = UpdateDevicePoint(FilterInfoLifeLitresUVField, float64(filLyfLtrUV), device)
	_ = UpdateDevicePoint(FilterInfoLifeMonthsUVField, float64(filLyfMnthUV), device)
	_ = UpdateDevicePoint(CO2LifeGramsField, float64(cO2LyfGrams), device)
	_ = UpdateDevicePoint(CO2LifeMonthsField, float64(cO2LyfMnths), device)
	_ = UpdateDevicePoint(CO2PressureField, float64(cO2Pressure), device)
	_ = UpdateDevicePoint(CO2TankCapacityField, float64(cO2TankCap), device)
	_ = UpdateDevicePoint(CO2AbsorptionRateField, float64(cO2AbsorpRate), device)
	_ = UpdateDevicePoint(SparklingFlowRateField, float64(sparklFlowRate), device)
	_ = UpdateDevicePoint(SparklingFlushTimeField, float64(sparklFlushTime), device)

	return nil
}

func pollPayloadDecoder(data []byte, device *model.Device) error {
	index := 1
	rebooted := (data[index]>>5)&1 == 1
	// sCov := (data[index]>>6)&1 == 1
	// wCov := (data[index]>>7)&1 == 1
	sms := int8((data[index]) & 0x3F)
	index += 1
	tempB := float32(binary.LittleEndian.Uint16(data[index:index+2])) / 10
	index += 2
	tempC := float32(binary.LittleEndian.Uint16(data[index:index+2])) / 10
	index += 2
	tempS := float32(binary.LittleEndian.Uint16(data[index:index+2])) / 10
	index += 2
	tempCond := float32(binary.LittleEndian.Uint16(data[index:index+2])) / 10
	index += 2
	f1 := data[index]
	index += 1
	f2 := data[index]
	index += 1
	f3 := data[index]
	index += 1
	f4 := data[index]
	index += 1
	kwh := float32(binary.LittleEndian.Uint32(data[index:index+4])) * 0.1
	index += 4
	dltDispB := int(binary.LittleEndian.Uint16(data[index : index+2]))
	index += 2
	dltDispC := int(binary.LittleEndian.Uint16(data[index : index+2]))
	index += 2
	dltDispS := int(binary.LittleEndian.Uint16(data[index : index+2]))
	index += 2
	dltLtrB := float32(binary.LittleEndian.Uint16(data[index:index+2])) / 10
	index += 2
	dltLtrC := float32(binary.LittleEndian.Uint16(data[index:index+2])) / 10
	index += 2
	dltLtrS := float32(binary.LittleEndian.Uint16(data[index:index+2])) / 10
	index += 2
	warningIndex := index
	fltrWrnInt := (data[index]>>0)&1 == 1
	fltrWrnExt := (data[index]>>1)&1 == 1
	index += 1
	fltrNfoUseLtrInt := int(binary.LittleEndian.Uint16(data[index : index+2]))
	index += 2
	fltrNfoUseDayInt := int(binary.LittleEndian.Uint16(data[index : index+2]))
	index += 2
	fltrNfoUseLtrExt := int(binary.LittleEndian.Uint16(data[index : index+2]))
	index += 2
	fltrNfoUseDayExt := int(binary.LittleEndian.Uint16(data[index : index+2]))
	index += 2

	fltrNfoUseLtrUV := 0
	fltrNfoUseDayUV := 0
	fltrWrnUV := false
	cO2GasPressureWrn := false
	cO2UsgGrams := 0
	cO2UsgDays := 0
	if data[0] >= 2 {
		fltrNfoUseLtrUV = int(binary.LittleEndian.Uint16(data[index : index+2]))
		index += 2
		fltrNfoUseDayUV = int(binary.LittleEndian.Uint16(data[index : index+2]))
		index += 2

		fltrWrnUV = (data[warningIndex]>>2)&1 == 1
		cO2GasPressureWrn = (data[warningIndex]>>3)&1 == 1
		cO2UsgGrams = int(binary.LittleEndian.Uint16(data[index : index+2]))
		index += 2
		cO2UsgDays = int(data[index])
		index += 1
	}

	_ = UpdateDevicePoint(RebootedField, utils.BoolToFloat(rebooted), device)
	_ = UpdateDevicePoint(SleepModeStatusField, float64(sms), device)
	_ = UpdateDevicePoint(TemperatureNTCBoilingField, float64(tempB), device)
	_ = UpdateDevicePoint(TemperatureNTCChilledField, float64(tempC), device)
	_ = UpdateDevicePoint(TemperatureNTCStreamField, float64(tempS), device)
	_ = UpdateDevicePoint(TemperatureNTCCondensorField, float64(tempCond), device)
	_ = UpdateDevicePoint(Fault1Field, float64(f1), device)
	_ = UpdateDevicePoint(Fault2Field, float64(f2), device)
	_ = UpdateDevicePoint(Fault3Field, float64(f3), device)
	_ = UpdateDevicePoint(Fault4Field, float64(f4), device)
	_ = UpdateDevicePoint(UsageEnergyKWhField, float64(kwh), device)
	_ = UpdateDevicePoint(UsageWaterDeltaDispensesBoilingField, float64(dltDispB), device)
	_ = UpdateDevicePoint(UsageWaterDeltaDispensesChilledField, float64(dltDispC), device)
	_ = UpdateDevicePoint(UsageWaterDeltaDispensesSparklingField, float64(dltDispS), device)
	_ = UpdateDevicePoint(UsageWaterDeltaLitresBoilingField, float64(dltLtrB), device)
	_ = UpdateDevicePoint(UsageWaterDeltaLitresChilledField, float64(dltLtrC), device)
	_ = UpdateDevicePoint(UsageWaterDeltaLitresSparklingField, float64(dltLtrS), device)
	_ = UpdateDevicePoint(FilterWarningInternalField, utils.BoolToFloat(fltrWrnInt), device)
	_ = UpdateDevicePoint(FilterWarningExternalField, utils.BoolToFloat(fltrWrnExt), device)
	_ = UpdateDevicePoint(FilterInfoUsageLitresInternalField, float64(fltrNfoUseLtrInt), device)
	_ = UpdateDevicePoint(FilterInfoUsageDaysInternalField, float64(fltrNfoUseDayInt), device)
	_ = UpdateDevicePoint(FilterInfoUsageLitresExternalField, float64(fltrNfoUseLtrExt), device)
	_ = UpdateDevicePoint(FilterInfoUsageDaysExternalField, float64(fltrNfoUseDayExt), device)
	// Pkt V2
	_ = UpdateDevicePoint(FilterInfoUsageLitresUVField, float64(fltrNfoUseLtrUV), device)
	_ = UpdateDevicePoint(FilterInfoUsageDaysUVField, float64(fltrNfoUseDayUV), device)
	_ = UpdateDevicePoint(FilterWarningUVField, utils.BoolToFloat(fltrWrnUV), device)
	_ = UpdateDevicePoint(CO2LowGasWarningField, utils.BoolToFloat(cO2GasPressureWrn), device)
	_ = UpdateDevicePoint(CO2UsageGramsField, float64(cO2UsgGrams), device)
	_ = UpdateDevicePoint(CO2UsageDaysField, float64(cO2UsgDays), device)

	return nil
}
