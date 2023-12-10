package schema

import (
	"github.com/NubeIO/lib-schema-go/schema"
)

type NetworkSchema struct {
	UUID           schema.UUID           `json:"uuid"`
	Name           schema.Name           `json:"name"`
	Description    schema.Description    `json:"description"`
	Enable         schema.Enable         `json:"enable"`
	PluginName     schema.PluginName     `json:"plugin_name"`
	SerialPort     SerialPortLora        `json:"serial_port"`
	SerialBaudRate schema.SerialBaudRate `json:"serial_baud_rate"`
	HistoryEnable  schema.HistoryEnable  `json:"history_enable"`
}

func GetNetworkSchema() *NetworkSchema {
	m := &NetworkSchema{}
	schema.Set(m)
	return m
}

type SerialPortLora struct {
	Type     string   `json:"type" default:"string"`
	Title    string   `json:"title" default:"Serial Port"`
	Options  []string `json:"enum" default:"[\"/dev/ttyAMA0\",\"/dev/ttyRS485-1\",\"/dev/ttyRS485-2\",\"/data/socat/loRa1\",\"/dev/ttyUSB0\",\"/dev/ttyUSB1\",\"/dev/ttyUSB2\",\"/dev/ttyUSB3\",\"/dev/ttyUSB4\",\"/data/socat/serialBridge1\",\"/dev/ttyACM0\"]"`
	EnumName []string `json:"enumNames" default:"[\"/dev/ttyAMA0\",\"/dev/ttyRS485-1\",\"/dev/ttyRS485-2\",\"/data/socat/loRa1\",\"/dev/ttyUSB0\",\"/dev/ttyUSB1\",\"/dev/ttyUSB2\",\"/dev/ttyUSB3\",\"/dev/ttyUSB4\",\"/data/socat/serialBridge1\",\"/dev/ttyACM0\"]"`
	Default  string   `json:"default" default:"/data/socat/LoRa1"`
	ReadOnly bool     `json:"readOnly" default:"false"`
}
