package decoder

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
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
	commonValues := &CommonValues{}
	decodeCommonValues(commonValues, data, devDesc.Model)
	if commonValues == nil {
		return errors.New("invalid common values")
	}

	updateDeviceFault(commonValues.ID, commonValues.Sensor, device.UUID, commonValues.Rssi)

	err := updateDevicePoint(RssiField, float64(commonValues.Rssi), device)
	if err != nil {
		return err
	}

	err = updateDevicePoint(SnrField, float64(commonValues.Snr), device)
	if err != nil {
		return err
	}

	bytes := getPayloadBytes(data)
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
	plID, _ := strconv.ParseInt(data[14:16], 16, 0)
	return TZHTPayloadType(plID)
}

func CheckPayloadLengthZHT(data string) bool {
	payloadLength := len(data) - 10 // removed addr, nonce and MAC
	payloadLength /= 2
	payloadType := getPayloadType(data)
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

func getPayloadBytes(data string) []byte {
	length, _ := strconv.ParseInt(data[12:14], 16, 0)
	bytes, _ := hex.DecodeString(data[16 : 16+((length-1)*2)])
	return bytes
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
// 	_ = updateDevicePoint("lora_firmware_major", float64(fwMa), device)
// 	_ = updateDevicePoint("lora_firmware_minor", float64(fwMi), device)
// 	_ = updateDevicePoint("lora_build_major", float64(buildMa), device)
// 	_ = updateDevicePoint("lora_build_minor", float64(buildMi), device)
// 	_ = updateDevicePoint("serial_number", sn, device)
// 	_ = updateDevicePoint("model_number", mn, device)
// 	_ = updateDevicePoint("product_number", pn, device)
// 	_ = updateDevicePoint("firmware_version", fw, device)
// 	_ = updateDevicePoint("calibration_date", calDate, device)
// 	_ = updateDevicePoint("first_50_litres_data", f50lDate, device)
// 	_ = updateDevicePoint("filter_log_date_internal", filtLogDateInt, device)
// 	_ = updateDevicePoint("filter_log_litres_internal", filtLogLitresInt, device)
// 	_ = updateDevicePoint("filter_log_date_external", filtLogDateExt, device)
// 	_ = updateDevicePoint("filter_log_litres_external", filtLogLitresExt, device)
// 	_ = updateDevicePoint("filter_log_date_uv", filtLogDateUV, device)
// 	_ = updateDevicePoint("filter_log_litres_uv", filtLogLitresUV, device)
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
		_ = updateDevicePoint(fmt.Sprintf("%s_%d", TimeStartField, i), float64(timeStart), device)
		_ = updateDevicePoint(fmt.Sprintf("%s_%d", TimeStopField, i), float64(timeStop), device)
		_ = updateDevicePoint(fmt.Sprintf("%s_%d", EnableStartField, i), utils.BoolToFloat(enableStart), device)
		_ = updateDevicePoint(fmt.Sprintf("%s_%d", EnableStopField, i), utils.BoolToFloat(enableStop), device)
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

	_ = updateDevicePoint(TimeField, float64(time), device)
	_ = updateDevicePoint(DispenseTimeBoilingField, float64(dispB), device)
	_ = updateDevicePoint(DispenseTimeChilledField, float64(dispC), device)
	_ = updateDevicePoint(DispenseTimeSparklingField, float64(dispS), device)
	_ = updateDevicePoint(TemperatureSPBoilingField, float64(tempSpB), device)
	_ = updateDevicePoint(TemperatureSPChilledField, float64(tempSpC), device)
	_ = updateDevicePoint(TemperatureSPSparklingField, float64(tempSpS), device)
	_ = updateDevicePoint(SleepModeSettingField, float64(sm), device)
	_ = updateDevicePoint(FilterInfoLifeLitresInternalField, float64(filLyfLtrInt), device)
	_ = updateDevicePoint(FilterInfoLifeMonthsInternalField, float64(filLyfMnthInt), device)
	_ = updateDevicePoint(FilterInfoLifeLitresExternalField, float64(filLyfLtrExt), device)
	_ = updateDevicePoint(FilterInfoLifeMonthsExternalField, float64(filLyfMnthExt), device)
	_ = updateDevicePoint(SafetyAllowTapChangesField, utils.BoolToFloat(sfTap), device)
	_ = updateDevicePoint(SafetyLockField, utils.BoolToFloat(sfL), device)
	_ = updateDevicePoint(SafetyHotIsolationField, utils.BoolToFloat(sfHi), device)
	_ = updateDevicePoint(SecurityEnableField, utils.BoolToFloat(secEn), device)
	_ = updateDevicePoint(SecurityPinField, float64(secPin), device)
	// Pkt V2
	_ = updateDevicePoint(FilterInfoLifeLitresUVField, float64(filLyfLtrUV), device)
	_ = updateDevicePoint(FilterInfoLifeMonthsUVField, float64(filLyfMnthUV), device)
	_ = updateDevicePoint(CO2LifeGramsField, float64(cO2LyfGrams), device)
	_ = updateDevicePoint(CO2LifeMonthsField, float64(cO2LyfMnths), device)
	_ = updateDevicePoint(CO2PressureField, float64(cO2Pressure), device)
	_ = updateDevicePoint(CO2TankCapacityField, float64(cO2TankCap), device)
	_ = updateDevicePoint(CO2AbsorptionRateField, float64(cO2AbsorpRate), device)
	_ = updateDevicePoint(SparklingFlowRateField, float64(sparklFlowRate), device)
	_ = updateDevicePoint(SparklingFlushTimeField, float64(sparklFlushTime), device)

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

	_ = updateDevicePoint(RebootedField, utils.BoolToFloat(rebooted), device)
	_ = updateDevicePoint(SleepModeStatusField, float64(sms), device)
	_ = updateDevicePoint(TemperatureNTCBoilingField, float64(tempB), device)
	_ = updateDevicePoint(TemperatureNTCChilledField, float64(tempC), device)
	_ = updateDevicePoint(TemperatureNTCStreamField, float64(tempS), device)
	_ = updateDevicePoint(TemperatureNTCCondensorField, float64(tempCond), device)
	_ = updateDevicePoint(Fault1Field, float64(f1), device)
	_ = updateDevicePoint(Fault2Field, float64(f2), device)
	_ = updateDevicePoint(Fault3Field, float64(f3), device)
	_ = updateDevicePoint(Fault4Field, float64(f4), device)
	_ = updateDevicePoint(UsageEnergyKWhField, float64(kwh), device)
	_ = updateDevicePoint(UsageWaterDeltaDispensesBoilingField, float64(dltDispB), device)
	_ = updateDevicePoint(UsageWaterDeltaDispensesChilledField, float64(dltDispC), device)
	_ = updateDevicePoint(UsageWaterDeltaDispensesSparklingField, float64(dltDispS), device)
	_ = updateDevicePoint(UsageWaterDeltaLitresBoilingField, float64(dltLtrB), device)
	_ = updateDevicePoint(UsageWaterDeltaLitresChilledField, float64(dltLtrC), device)
	_ = updateDevicePoint(UsageWaterDeltaLitresSparklingField, float64(dltLtrS), device)
	_ = updateDevicePoint(FilterWarningInternalField, utils.BoolToFloat(fltrWrnInt), device)
	_ = updateDevicePoint(FilterWarningExternalField, utils.BoolToFloat(fltrWrnExt), device)
	_ = updateDevicePoint(FilterInfoUsageLitresInternalField, float64(fltrNfoUseLtrInt), device)
	_ = updateDevicePoint(FilterInfoUsageDaysInternalField, float64(fltrNfoUseDayInt), device)
	_ = updateDevicePoint(FilterInfoUsageLitresExternalField, float64(fltrNfoUseLtrExt), device)
	_ = updateDevicePoint(FilterInfoUsageDaysExternalField, float64(fltrNfoUseDayExt), device)
	// Pkt V2
	_ = updateDevicePoint(FilterInfoUsageLitresUVField, float64(fltrNfoUseLtrUV), device)
	_ = updateDevicePoint(FilterInfoUsageDaysUVField, float64(fltrNfoUseDayUV), device)
	_ = updateDevicePoint(FilterWarningUVField, utils.BoolToFloat(fltrWrnUV), device)
	_ = updateDevicePoint(CO2LowGasWarningField, utils.BoolToFloat(cO2GasPressureWrn), device)
	_ = updateDevicePoint(CO2UsageGramsField, float64(cO2UsgGrams), device)
	_ = updateDevicePoint(CO2UsageDaysField, float64(cO2UsgDays), device)

	return nil
}
