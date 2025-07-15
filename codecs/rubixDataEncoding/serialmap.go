package rubixDataEncoding

const DATA_TYPE_BIT_COUNT = 6

type DataType int
type MetaDataKey int
type BIT_TYPE uint64

type MetaData struct {
	dataType     DataType
	lowValue     int
	highValue    int
	decimalPoint int
	byteCount    int
}

const (
	MDK_TEMP             MetaDataKey = 1
	MDK_RH               MetaDataKey = 2
	MDK_LUX              MetaDataKey = 3
	MDK_MOVEMENT         MetaDataKey = 4
	MDK_COUNTER          MetaDataKey = 5
	MDK_DIGITAL          MetaDataKey = 6
	MDK_VOLTAGE_0_10     MetaDataKey = 7
	MDK_MILLIAMPS_4_20   MetaDataKey = 8
	MDK_OHM              MetaDataKey = 10
	MDK_CO2              MetaDataKey = 11
	MDK_BATTERY_VOLTAGE  MetaDataKey = 12
	MDK_PUSH_FREQUENCY   MetaDataKey = 13
	MDK_ANALOG_IN        MetaDataKey = 16
	MDK_FIRMWARE_VERSION MetaDataKey = 61
	MDK_HARDWARE_VERSION MetaDataKey = 62
	MDK_UINT_8           MetaDataKey = 30
	MDK_INT_8            MetaDataKey = 31
	MDK_UINT_16          MetaDataKey = 32
	MDK_INT_16           MetaDataKey = 33
	MDK_UINT_32          MetaDataKey = 34
	MDK_INT_32           MetaDataKey = 35
	MDK_UINT_64          MetaDataKey = 36
	MDK_INT_64           MetaDataKey = 37
	MDK_BOOL             MetaDataKey = 38
	MDK_CHAR             MetaDataKey = 39
	MDK_FLOAT            MetaDataKey = 40
	MDK_DOUBLE           MetaDataKey = 41
	MDK_STRING           MetaDataKey = 42
	MDK_ERROR            MetaDataKey = 43
)

const (
	FIXEDPOINT = 1
	DATAPOINT  = 2
)

var serialMap = map[MetaDataKey]MetaData{
	MDK_TEMP:             {FIXEDPOINT, -45, 120, 2, 0},
	MDK_RH:               {FIXEDPOINT, 0, 100, 2, 0},
	MDK_LUX:              {FIXEDPOINT, 0, 65534, 0, 0},
	MDK_MOVEMENT:         {FIXEDPOINT, 0, 1, 0, 0},
	MDK_COUNTER:          {FIXEDPOINT, 0, 1048576, 0, 0},
	MDK_DIGITAL:          {FIXEDPOINT, 0, 1, 0, 0},
	MDK_VOLTAGE_0_10:     {FIXEDPOINT, 0, 10, 2, 0},
	MDK_MILLIAMPS_4_20:   {FIXEDPOINT, 4, 20, 2, 0},
	MDK_OHM:              {FIXEDPOINT, 0, 1048576, 0, 0},
	MDK_CO2:              {FIXEDPOINT, 0, 400, 0, 0},
	MDK_BATTERY_VOLTAGE:  {FIXEDPOINT, 0, 6, 1, 0},
	MDK_PUSH_FREQUENCY:   {FIXEDPOINT, 0, 2000, 0, 0},
	MDK_FIRMWARE_VERSION: {FIXEDPOINT, 0, 255, 0, 0},
	MDK_HARDWARE_VERSION: {FIXEDPOINT, 0, 255, 0, 0},
	MDK_UINT_8:           {DATAPOINT, 0, 0, 0, 1},
	MDK_INT_8:            {DATAPOINT, 0, 0, 0, 1},
	MDK_UINT_16:          {DATAPOINT, 0, 0, 0, 2},
	MDK_INT_16:           {DATAPOINT, 0, 0, 0, 2},
	MDK_UINT_32:          {DATAPOINT, 0, 0, 0, 4},
	MDK_INT_32:           {DATAPOINT, 0, 0, 0, 4},
	MDK_UINT_64:          {DATAPOINT, 0, 0, 0, 8},
	MDK_INT_64:           {DATAPOINT, 0, 0, 0, 8},
	MDK_BOOL:             {FIXEDPOINT, 0, 1, 0, 0},
	MDK_CHAR:             {DATAPOINT, 0, 0, 0, 1},
	MDK_FLOAT:            {DATAPOINT, 0, 0, 0, 4},
	MDK_DOUBLE:           {DATAPOINT, 0, 0, 0, 8},
	MDK_ERROR:            {DATAPOINT, 0, 0, 0, 1},
}

const (
	TempField           = "temp"
	RHField             = "rh"
	LuxField            = "lux"
	MovementField       = "movement"
	CounterField        = "count"
	DigitalField        = "digital"
	VoltageField        = "0-10v"
	MilliampsField      = "4-20ma"
	OhmField            = "ohm"
	CO2Field            = "co2"
	BatteryVoltageField = "battery-voltage"
	PushFrequencyField  = "push-frequency"
	AnalogInField       = "ai-raw"
	FwVersionField      = "firmware-version"
	HwVersionField      = "hardware-version"
	BoolField           = "bool"
	CharField           = "char"
	UInt8Field          = "uint_8"
	Int8Field           = "int_8"
	UInt16Field         = "uint_16"
	Int16Field          = "int_16"
	UInt32Field         = "uint_32"
	Int32Field          = "int_32"
	UInt64Field         = "uint_64"
	Int64Field          = "int_64"
	FloatField          = "float"
	DoubleField         = "double"
	ErrorField          = "error"
)

func getMetaData(metaDataKey MetaDataKey) MetaData {
	return serialMap[metaDataKey]
}

func (m MetaDataKey) String() string {
	switch m {
	case MDK_TEMP:
		return TempField
	case MDK_RH:
		return RHField
	case MDK_LUX:
		return LuxField
	case MDK_MOVEMENT:
		return MovementField
	case MDK_COUNTER:
		return CounterField
	case MDK_DIGITAL:
		return DigitalField
	case MDK_VOLTAGE_0_10:
		return VoltageField
	case MDK_MILLIAMPS_4_20:
		return MilliampsField
	case MDK_OHM:
		return OhmField
	case MDK_CO2:
		return CO2Field
	case MDK_BATTERY_VOLTAGE:
		return BatteryVoltageField
	case MDK_PUSH_FREQUENCY:
		return PushFrequencyField
	case MDK_ANALOG_IN:
		return AnalogInField
	case MDK_FIRMWARE_VERSION:
		return FwVersionField
	case MDK_HARDWARE_VERSION:
		return HwVersionField
	case MDK_BOOL:
		return BoolField
	case MDK_CHAR:
		return CharField
	case MDK_UINT_8:
		return UInt8Field
	case MDK_INT_8:
		return Int8Field
	case MDK_UINT_16:
		return UInt16Field
	case MDK_INT_16:
		return Int16Field
	case MDK_UINT_32:
		return UInt32Field
	case MDK_INT_32:
		return Int32Field
	case MDK_UINT_64:
		return UInt64Field
	case MDK_INT_64:
		return Int64Field
	case MDK_FLOAT:
		return FloatField
	case MDK_DOUBLE:
		return DoubleField
	case MDK_ERROR:
		return ErrorField
	default:
		return "unknown"
	}
}
