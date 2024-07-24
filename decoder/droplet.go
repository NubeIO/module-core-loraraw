package decoder

import (
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
	// TODO: Will this still be valid?
	return dl == 36 || dl == 32 || dl == 44
}

func DecodeDropletTH(data string, devDesc *LoRaDeviceDescription, device *model.Device) error {
	updateDeviceFault(devDesc.Model, device.UUID)

	temperature, err := dropletTemp(data)
	if err != nil {
		return err
	}
	pressure, err := dropletPressure(data)
	if err != nil {
		return err
	}
	humidity, err := dropletHumidity(data)
	if err != nil {
		return err
	}
	voltage, err := dropletVoltage(data)
	if err != nil {
		return err
	}

	_ = UpdateDevicePoint(TemperatureField, temperature, device)
	_ = UpdateDevicePoint(PressureField, pressure, device)
	_ = UpdateDevicePoint(HumidityField, float64(humidity), device)
	_ = UpdateDevicePoint(DropletVoltageField, voltage, device)

	return nil
}

func DecodeDropletTHL(data string, devDesc *LoRaDeviceDescription, device *model.Device) error {
	err := DecodeDropletTH(data, devDesc, device)
	if err != nil {
		return err
	}
	light, err := dropletLight(data)
	if err != nil {
		return err
	}
	_ = UpdateDevicePoint(LightField, float64(light), device)
	return nil
}

func DecodeDropletTHLM(data string, devDesc *LoRaDeviceDescription, device *model.Device) error {
	err := DecodeDropletTHL(data, devDesc, device)
	if err != nil {
		return err
	}
	motion, err := dropletMotion(data)
	if err != nil {
		return err
	}
	_ = UpdateDevicePoint(MotionField, utils.BoolToFloat(motion), device)
	return nil
}

func dropletTemp(data string) (float64, error) {
	v, err := strconv.ParseInt(data[2:4]+data[:2], 16, 0)
	if err != nil {
		return 0, err
	}
	v_ := float64(v) / 100
	return v_, nil
}

func dropletPressure(data string) (float64, error) {
	v, err := strconv.ParseInt(data[6:8]+data[4:6], 16, 0)
	if err != nil {
		return 0, err
	}
	v_ := float64(v) / 10
	return v_, err
}

func dropletHumidity(data string) (int, error) {
	v, err := strconv.ParseInt(data[8:10], 16, 0)
	if err != nil {
		return 0, err
	}
	v = v & 127
	return int(v), nil
}

func dropletVoltage(data string) (float64, error) {
	v, err := strconv.ParseInt(data[14:16], 16, 0)
	if err != nil {
		return 0, err
	}
	v_ := float64(v) / 50
	if v_ < 1 { // added in by aidan not tested asked by Craig (its needed when the droplet uses lithium batteries)
		v_ = v_ - 0.06 + 5
	}
	return v_, nil
}

func dropletLight(data string) (int, error) {
	v, err := strconv.ParseInt(data[12:14]+data[10:12], 16, 0)
	if err != nil {
		return 0, err
	}
	return int(v), nil
}

func dropletMotion(data string) (bool, error) {
	v, err := strconv.ParseInt(data[8:10], 16, 0)
	if err != nil {
		return false, err
	}
	return v > 127, nil
}
