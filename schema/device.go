package schema

import "github.com/NubeIO/lib-schema-go/schema"

const (
	DeviceModelTHLM         = "THLM"
	DeviceModelTHL          = "THL"
	DeviceModelTH           = "TH"
	DeviceModelMicroEdgeV1  = "MicroEdgeV1"
	DeviceModelMicroEdgeV2  = "MicroEdgeV2"
	DeviceModelZiptHydroTap = "ZipHydroTap"
	DeviceModelRubix        = "Rubix"
)

type DeviceKey struct {
	Type  string `json:"type" default:"string"`
	Title string `json:"title" default:"Device Key"`
	Min   int    `json:"minLength" default:"32"`
	Max   int    `json:"maxLength" default:"32"`
}

type DeviceSchema struct {
	UUID          schema.UUID                     `json:"uuid"`
	Name          schema.Name                     `json:"name"`
	Description   schema.Description              `json:"description"`
	Enable        schema.Enable                   `json:"enable"`
	AddressUUID   schema.AddressUUID              `json:"address_uuid"`
	Model         schema.Model                    `json:"model"`
	HistoryEnable schema.HistoryEnableDefaultTrue `json:"history_enable"`
	DeviceKey     DeviceKey                       `json:"manufacture" type:"secret"`
}

func GetDeviceSchema() *DeviceSchema {
	models := []string{DeviceModelTHLM, DeviceModelTHL, DeviceModelTH, DeviceModelMicroEdgeV1, DeviceModelMicroEdgeV2, DeviceModelZiptHydroTap, DeviceModelRubix}
	m := &DeviceSchema{}
	m.AddressUUID.Min = 8
	m.AddressUUID.Max = 8
	m.Model.EnumName = models
	m.Model.Options = models
	schema.Set(m)
	return m
}
