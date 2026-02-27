package legacyDecoders

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/NubeIO/lib-utils-go/nstring"
	"github.com/NubeIO/module-core-loraraw/codec"
	"github.com/NubeIO/module-core-loraraw/utils"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
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

func DecodeZHT(
	data string,
	_ []byte,
	_ *codec.LoRaDeviceDescription,
	device *model.Device,
	updatePointFn codec.UpdateDevicePointFunc,
	updatePointErrFn codec.UpdateDevicePointErrorFunc,
	updateDeviceMetaTagsFn codec.UpdateDeviceMetaTagsFunc,
) error {
	bytes, err := getPayloadBytes(data)
	if err != nil {
		return err
	}

	switch pl := getPayloadType(data); pl {
	case StaticData:
		return staticPayloadDecoder(bytes, device, updateDeviceMetaTagsFn)
	case WriteData:
		err := writePayloadDecoder(bytes, device, updatePointFn)
		return err
	case PollData:
		err := pollPayloadDecoder(bytes, device, updatePointFn)
		return err
	}

	return nil
}

func getPayloadBytes(data string) ([]byte, error) {
	if len(data) < 2 {
		return nil, fmt.Errorf("payload too short")
	}
	bytes, err := hex.DecodeString(data[2:])
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

func getPayloadType(data string) TZHTPayloadType {
	if len(data) < 2 {
		return ErrorData
	}
	plID, _ := strconv.ParseInt(data[:2], 16, 0)
	return TZHTPayloadType(plID)
}

func getPacketVersion(data string) uint8 {
	if len(data) < 4 {
		return 0
	}
	v, err := strconv.ParseInt(data[2:4], 16, 0)
	if err != nil {
		return 0
	}
	return uint8(v)
}

func CheckPayloadLengthZHT(data string) bool {
	dataLength := len(data) / 2
	payloadType := getPayloadType(data)

	if getPacketVersion(data) == 1 {
		return (payloadType == StaticData && dataLength == ZHTPlLenStaticV1) ||
			(payloadType == WriteData && dataLength == ZHTPlLenWriteV1) ||
			(payloadType == PollData && dataLength == ZHTPlLenPollV1)
	} else if getPacketVersion(data) == 2 {
		return (payloadType == StaticData && dataLength == ZHTPlLenStaticV2) ||
			(payloadType == WriteData && dataLength == ZHTPlLenWriteV2) ||
			(payloadType == PollData && dataLength == ZHTPlLenPollV2)
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
	commonValueFields := codec.GetCommonValueNames()
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
	if len(bytes) < 3 {
		return ""
	}
	return fmt.Sprintf("%d/%d/%d", bytes[0], bytes[1], bytes[2])
}

// No usages of staticPayloadDecoder method
func staticPayloadDecoder(data []byte, device *model.Device, updateDeviceMetaTagsFn codec.UpdateDeviceMetaTagsFunc) error {
	// Validate minimum payload length (version 1: 89 bytes, version 2: 94 bytes)
	minLength := 96
	if len(data) < minLength {
		return fmt.Errorf("static payload too short: required>=%d actual=%d", minLength, len(data))
	}

	index := 1
	fwMa := int(data[index])
	index += 1
	fwMi := int(data[index])
	index += 1
	buildMa := int(data[index])
	index += 1
	buildMi := int(data[index])
	index += 1
	sn := bytesToString(data[index : index+15])
	index += 15
	mn := bytesToString(data[index : index+20])
	index += 20
	pn := bytesToString(data[index : index+20])
	index += 20
	fw := bytesToString(data[index : index+20])
	index += 20
	calDate := bytesToDate(data[index : index+3])
	index += 3
	f50lDate := bytesToDate(data[index : index+3])
	index += 3
	filtLogDateInt := bytesToDate(data[index : index+3])
	index += 3
	filtLogLitresInt := int(binary.LittleEndian.Uint16(data[index : index+2]))
	index += 2
	filtLogDateExt := bytesToDate(data[index : index+3])
	index += 3
	filtLogLitresExt := int(binary.LittleEndian.Uint16(data[index : index+2]))
	index += 2

	filtLogDateUV := ""
	filtLogLitresUV := 0
	if data[0] >= 2 {
		// Check if we have enough data for v2 fields
		if index+5 > len(data) {
			return fmt.Errorf("insufficient data for v2 static payload fields: index=%d len=%d", index, len(data))
		}
		filtLogDateUV = bytesToDate(data[index : index+3])
		index += 3
		filtLogLitresUV = int(binary.LittleEndian.Uint16(data[index : index+2]))
		index += 2
	}

	addressUUID := nstring.DerefString(device.AddressUUID)
	var modbusAddress int64
	if len(addressUUID) >= 6 {
		modbusAddress, _ = strconv.ParseInt(addressUUID[4:6], 16, 0)
	}

	metaTags := map[string]string{
		"lora_firmware_major":        strconv.Itoa(fwMa),
		"lora_firmware_minor":        strconv.Itoa(fwMi),
		"lora_build_major":           strconv.Itoa(buildMa),
		"lora_build_minor":           strconv.Itoa(buildMi),
		"serial_number":              sn,
		"model_number":               mn,
		"product_number":             pn,
		"firmware_version":           fw,
		"calibration_date":           calDate,
		"first_50_litres_data":       f50lDate,
		"filter_log_date_internal":   filtLogDateInt,
		"filter_log_litres_internal": strconv.Itoa(filtLogLitresInt),
		"filter_log_date_external":   filtLogDateExt,
		"filter_log_litres_external": strconv.Itoa(filtLogLitresExt),
		"filter_log_date_uv":         filtLogDateUV,
		"filter_log_litres_uv":       strconv.Itoa(filtLogLitresUV),
		"modbus_address":             strconv.Itoa(int(modbusAddress)),
	}

	for k, v := range metaTags {
		for _, metaTag := range device.MetaTags {
			if k == metaTag.Key {
				metaTag.Value = v
				continue
			}
		}
		device.MetaTags = append(device.MetaTags, &model.DeviceMetaTag{
			DeviceUUID: device.UUID,
			Key:        k,
			Value:      v,
		})
	}
	return updateDeviceMetaTagsFn(device.UUID, device.MetaTags)
}

func writePayloadDecoder(data []byte, device *model.Device, updatePointFn codec.UpdateDevicePointFunc) error {
	minLength := 22 + (ZipHTTimerLength * 4)
	if len(data) < minLength {
		return fmt.Errorf("write payload too short: required>=%d actual=%d", minLength, len(data))
	}

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
		_ = updatePointFn(fmt.Sprintf("%s_%d", TimeStartField, i), float64(timeStart), device, nil)
		_ = updatePointFn(fmt.Sprintf("%s_%d", TimeStopField, i), float64(timeStop), device, nil)
		_ = updatePointFn(fmt.Sprintf("%s_%d", EnableStartField, i), utils.BoolToFloat(enableStart), device, nil)
		_ = updatePointFn(fmt.Sprintf("%s_%d", EnableStopField, i), utils.BoolToFloat(enableStop), device, nil)
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
		if index+15 > len(data) {
			return fmt.Errorf("insufficient data for v2 write payload fields: index=%d len=%d", index, len(data))
		}
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

	_ = updatePointFn(TimeField, float64(time), device, nil)
	_ = updatePointFn(DispenseTimeBoilingField, float64(dispB), device, nil)
	_ = updatePointFn(DispenseTimeChilledField, float64(dispC), device, nil)
	_ = updatePointFn(DispenseTimeSparklingField, float64(dispS), device, nil)
	_ = updatePointFn(TemperatureSPBoilingField, float64(tempSpB), device, nil)
	_ = updatePointFn(TemperatureSPChilledField, float64(tempSpC), device, nil)
	_ = updatePointFn(TemperatureSPSparklingField, float64(tempSpS), device, nil)
	_ = updatePointFn(SleepModeSettingField, float64(sm), device, nil)
	_ = updatePointFn(FilterInfoLifeLitresInternalField, float64(filLyfLtrInt), device, nil)
	_ = updatePointFn(FilterInfoLifeMonthsInternalField, float64(filLyfMnthInt), device, nil)
	_ = updatePointFn(FilterInfoLifeLitresExternalField, float64(filLyfLtrExt), device, nil)
	_ = updatePointFn(FilterInfoLifeMonthsExternalField, float64(filLyfMnthExt), device, nil)
	_ = updatePointFn(SafetyAllowTapChangesField, utils.BoolToFloat(sfTap), device, nil)
	_ = updatePointFn(SafetyLockField, utils.BoolToFloat(sfL), device, nil)
	_ = updatePointFn(SafetyHotIsolationField, utils.BoolToFloat(sfHi), device, nil)
	_ = updatePointFn(SecurityEnableField, utils.BoolToFloat(secEn), device, nil)
	_ = updatePointFn(SecurityPinField, float64(secPin), device, nil)
	// Pkt V2
	_ = updatePointFn(FilterInfoLifeLitresUVField, float64(filLyfLtrUV), device, nil)
	_ = updatePointFn(FilterInfoLifeMonthsUVField, float64(filLyfMnthUV), device, nil)
	_ = updatePointFn(CO2LifeGramsField, float64(cO2LyfGrams), device, nil)
	_ = updatePointFn(CO2LifeMonthsField, float64(cO2LyfMnths), device, nil)
	_ = updatePointFn(CO2PressureField, float64(cO2Pressure), device, nil)
	_ = updatePointFn(CO2TankCapacityField, float64(cO2TankCap), device, nil)
	_ = updatePointFn(CO2AbsorptionRateField, float64(cO2AbsorpRate), device, nil)
	_ = updatePointFn(SparklingFlowRateField, float64(sparklFlowRate), device, nil)
	_ = updatePointFn(SparklingFlushTimeField, float64(sparklFlushTime), device, nil)

	return nil
}

func pollPayloadDecoder(data []byte, device *model.Device, updatePointFn codec.UpdateDevicePointFunc) error {
	minLength := 39
	if len(data) < minLength {
		return fmt.Errorf("poll payload too short: required>=%d actual=%d", minLength, len(data))
	}

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
		if index+7 > len(data) {
			return fmt.Errorf("insufficient data for v2 poll payload fields: index=%d len=%d", index, len(data))
		}
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

	_ = updatePointFn(RebootedField, utils.BoolToFloat(rebooted), device, nil)
	_ = updatePointFn(SleepModeStatusField, float64(sms), device, nil)
	_ = updatePointFn(TemperatureNTCBoilingField, float64(tempB), device, nil)
	_ = updatePointFn(TemperatureNTCChilledField, float64(tempC), device, nil)
	_ = updatePointFn(TemperatureNTCStreamField, float64(tempS), device, nil)
	_ = updatePointFn(TemperatureNTCCondensorField, float64(tempCond), device, nil)
	_ = updatePointFn(Fault1Field, float64(f1), device, nil)
	_ = updatePointFn(Fault2Field, float64(f2), device, nil)
	_ = updatePointFn(Fault3Field, float64(f3), device, nil)
	_ = updatePointFn(Fault4Field, float64(f4), device, nil)
	_ = updatePointFn(UsageEnergyKWhField, float64(kwh), device, nil)
	_ = updatePointFn(UsageWaterDeltaDispensesBoilingField, float64(dltDispB), device, nil)
	_ = updatePointFn(UsageWaterDeltaDispensesChilledField, float64(dltDispC), device, nil)
	_ = updatePointFn(UsageWaterDeltaDispensesSparklingField, float64(dltDispS), device, nil)
	_ = updatePointFn(UsageWaterDeltaLitresBoilingField, float64(dltLtrB), device, nil)
	_ = updatePointFn(UsageWaterDeltaLitresChilledField, float64(dltLtrC), device, nil)
	_ = updatePointFn(UsageWaterDeltaLitresSparklingField, float64(dltLtrS), device, nil)
	_ = updatePointFn(FilterWarningInternalField, utils.BoolToFloat(fltrWrnInt), device, nil)
	_ = updatePointFn(FilterWarningExternalField, utils.BoolToFloat(fltrWrnExt), device, nil)
	_ = updatePointFn(FilterInfoUsageLitresInternalField, float64(fltrNfoUseLtrInt), device, nil)
	_ = updatePointFn(FilterInfoUsageDaysInternalField, float64(fltrNfoUseDayInt), device, nil)
	_ = updatePointFn(FilterInfoUsageLitresExternalField, float64(fltrNfoUseLtrExt), device, nil)
	_ = updatePointFn(FilterInfoUsageDaysExternalField, float64(fltrNfoUseDayExt), device, nil)
	// Pkt V2
	_ = updatePointFn(FilterInfoUsageLitresUVField, float64(fltrNfoUseLtrUV), device, nil)
	_ = updatePointFn(FilterInfoUsageDaysUVField, float64(fltrNfoUseDayUV), device, nil)
	_ = updatePointFn(FilterWarningUVField, utils.BoolToFloat(fltrWrnUV), device, nil)
	_ = updatePointFn(CO2LowGasWarningField, utils.BoolToFloat(cO2GasPressureWrn), device, nil)
	_ = updatePointFn(CO2UsageGramsField, float64(cO2UsgGrams), device, nil)
	_ = updatePointFn(CO2UsageDaysField, float64(cO2UsgDays), device, nil)

	return nil
}
