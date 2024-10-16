package endec

const HEADER_BIT_COUNT = 6

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
	MDK_TEMP             = 1
	MDK_RH               = 2
	MDK_LUX              = 3
	MDK_MOVEMENT         = 4
	MDK_COUNTER          = 5
	MDK_DIGITAL          = 6
	MDK_VOLTAGE_0_10     = 7
	MDK_MILLIAMPS_4_20   = 8
	MDK_OHM              = 10
	MDK_CO2              = 11
	MDK_BATTERY_VOLTAGE  = 12
	MDK_PUSH_FREQUENCY   = 13
	MDK_RAW              = 16
	MDK_UO               = 17
	MDK_UI               = 18
	MDK_DO               = 19
	MDK_DI               = 20
	MDK_FIRMWARE_VERSION = 61
	MDK_HARDWARE_VERSION = 62
	MDK_UINT_8           = 30
	MDK_INT_8            = 31
	MDK_UINT_16          = 32
	MDK_INT_16           = 33
	MDK_UINT_32          = 34
	MDK_INT_32           = 35
	MDK_UINT_64          = 36
	MDK_INT_64           = 37
	MDK_BOOL             = 38
	MDK_CHAR             = 39
	MDK_FLOAT            = 40
	MDK_DOUBLE           = 41
	MDK_STRING           = 42
	MDK_ERROR            = 43
)

const (
	FIXEDPOINT = 1
	DATAPOINT  = 2
)

var serialMap = map[int]MetaData{
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
	MDK_RAW:              {FIXEDPOINT, 0, 1, 3, 0},
	MDK_UO:               {FIXEDPOINT, 0, 1, 3, 0},
	MDK_UI:               {FIXEDPOINT, 0, 1, 3, 0},
	MDK_DO:               {FIXEDPOINT, 0, 1, 0, 0},
	MDK_DI:               {FIXEDPOINT, 0, 1, 0, 0},
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
}

func getMetaData(header MetaDataKey) MetaData {
	return serialMap[int(header)]
}
