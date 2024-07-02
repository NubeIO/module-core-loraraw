package decoder

import (
	"errors"
	"github.com/NubeIO/module-core-loraraw/utils"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
	"strconv"
)

const (
	DropletVoltageField = "voltage"
	TemperatureField    = "temperature"
	PressureField       = "pressure"
	HumidityField       = "humidity"
	LightField          = "light"
	MotionField         = "motion"
)

func GetTHPointNames() []string {
	commonValueFields := GetCommonValueNames()
	dropletTHFields := []string{
		DropletVoltageField,
		TemperatureField,
		PressureField,
		HumidityField,
	}
	return append(commonValueFields, dropletTHFields...)
}

func GetTHLPointNames() []string {
	dropletTHFields := GetTHPointNames()
	return append(dropletTHFields, LightField)
}

func GetTHLMPointNames() []string {
	dropletTHLFields := GetTHLPointNames()
	return append(dropletTHLFields, MotionField)
}

func CheckPayloadLengthDroplet(data string) bool {
	dl := len(data)
	return dl == 36 || dl == 32 || dl == 44
}

func DecodeDropletTH(data string, devDesc *LoRaDeviceDescription, device *model.Device) error {
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

	temperature := dropletTemp(data)
	pressure := dropletPressure(data)
	humidity := dropletHumidity(data)
	voltage := dropletVoltage(data)

	_ = updateDevicePoint(TemperatureField, temperature, device)
	_ = updateDevicePoint(PressureField, pressure, device)
	_ = updateDevicePoint(HumidityField, float64(humidity), device)
	_ = updateDevicePoint(DropletVoltageField, voltage, device)

	return nil
}

func DecodeDropletTHL(data string, devDesc *LoRaDeviceDescription, device *model.Device) error {
	err := DecodeDropletTH(data, devDesc, device)
	if err != nil {
		return err
	}

	light := dropletLight(data)
	_ = updateDevicePoint(LightField, float64(light), device)
	return nil
}

func DecodeDropletTHLM(data string, devDesc *LoRaDeviceDescription, device *model.Device) error {
	err := DecodeDropletTHL(data, devDesc, device)
	if err != nil {
		return err
	}

	motion := dropletMotion(data)
	_ = updateDevicePoint(MotionField, utils.BoolToFloat(motion), device)
	return nil
}

func dropletTemp(data string) float64 {
	v, _ := strconv.ParseInt(data[10:12]+data[8:10], 16, 0)
	v_ := float64(v) / 100
	return v_
}

func dropletPressure(data string) float64 {
	v, _ := strconv.ParseInt(data[14:16]+data[12:14], 16, 0)
	v_ := float64(v) / 10
	return v_
}

func dropletHumidity(data string) int {
	v, _ := strconv.ParseInt(data[16:18], 16, 0)
	v = v & 127
	return int(v)
}

func dropletVoltage(data string) float64 {
	v, _ := strconv.ParseInt(data[22:24], 16, 0)
	v_ := float64(v) / 50
	if v_ < 1 { // added in by aidan not tested asked by Craig (its needed when the droplet uses lithium batteries)
		v_ = v_ - 0.06 + 5
	}
	return v_
}

func dropletLight(data string) int {
	v, _ := strconv.ParseInt(data[20:22]+data[18:20], 16, 0)
	return int(v)
}

func dropletMotion(data string) bool {
	v, _ := strconv.ParseInt(data[16:18], 16, 0)
	return v > 127
}
