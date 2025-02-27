package schema

type DataType struct {
	Type     string   `json:"type" default:"string"`
	Title    string   `json:"title" default:"Data Type"`
	Options  []string `json:"enum" default:"[\"1\", \"2\", \"3\", \"4\", \"5\", \"6\", \"7\", \"8\", \"10\", \"11\", \"12\", \"13\", \"16\", \"17\", \"18\", \"19\", \"20\", \"30\", \"31\", \"32\", \"33\", \"34\", \"35\", \"36\", \"37\", \"38\", \"39\", \"40\", \"41\", \"42\", \"61\", \"62\"]"`
	EnumName []string `json:"enumNames" default:"[\"MDK_TEMP\", \"MDK_RH\", \"MDK_LUX\", \"MDK_MOVEMENT\", \"MDK_COUNTER\", \"MDK_DIGITAL\", \"MDK_VOLTAGE_0_10\", \"MDK_MILLIAMPS_4_20\", \"MDK_OHM\", \"MDK_CO2\", \"MDK_BATTERY_VOLTAGE\", \"MDK_PUSH_FREQUENCY\", \"MDK_RAW\", \"MDK_UO\", \"MDK_UI\", \"MDK_DO\", \"MDK_DI\", \"MDK_UINT_8\", \"MDK_INT_8\", \"MDK_UINT_16\", \"MDK_INT_16\", \"MDK_UINT_32\", \"MDK_INT_32\", \"MDK_UINT_64\", \"MDK_INT_64\", \"MDK_BOOL\", \"MDK_CHAR\", \"MDK_FLOAT\", \"MDK_DOUBLE\", \"MDK_STRING\", \"MDK_FIRMWARE_VERSION\", \"MDK_HARDWARE_VERSION\"]"`
	Default  string   `json:"default" default:"40"`
	ReadOnly bool     `json:"readOnly" default:"false"`
}
