package legacyDecoders

import (
	"fmt"
	"strconv"

	"github.com/NubeIO/module-core-loraraw/codec"
	"github.com/NubeIO/module-core-loraraw/utils"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
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
	commonValueFields := codec.GetCommonValueNames()
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

func DecodeDropletTH(
	data string,
	_ []byte,
	_ *codec.LoRaDeviceDescription,
	device *model.Device,
	updatePointFn codec.UpdateDevicePointFunc,
	updatePointErrFn codec.UpdateDevicePointErrorFunc,
	_ codec.UpdateDeviceMetaTagsFunc,
) error {
	temperature, err := dropletTemp(data)
	if err != nil {
		return updatePointErrFn(TemperatureField, err, device, nil)
	}
	pressure, err := dropletPressure(data)
	if err != nil {
		return updatePointErrFn(PressureField, err, device, nil)
	}
	humidity, err := dropletHumidity(data)
	if err != nil {
		return updatePointErrFn(HumidityField, err, device, nil)
	}
	voltage, err := dropletVoltage(data)
	if err != nil {
		return updatePointErrFn(DropletVoltageField, err, device, nil)
	}

	_ = updatePointFn(TemperatureField, temperature, device, nil)
	_ = updatePointFn(PressureField, pressure, device, nil)
	_ = updatePointFn(HumidityField, float64(humidity), device, nil)
	_ = updatePointFn(DropletVoltageField, voltage, device, nil)

	return nil
}

func DecodeDropletTHL(
	data string,
	dataBytes []byte,
	devDesc *codec.LoRaDeviceDescription,
	device *model.Device,
	updatePointFn codec.UpdateDevicePointFunc,
	updatePointErrFn codec.UpdateDevicePointErrorFunc,
	updateDeviceMetaTagFn codec.UpdateDeviceMetaTagsFunc,
) error {
	err := DecodeDropletTH(
		data,
		dataBytes,
		devDesc,
		device,
		updatePointFn,
		updatePointErrFn,
		updateDeviceMetaTagFn,
	)
	if err != nil {
		return err
	}
	light, err := dropletLight(data)
	if err != nil {
		return updatePointErrFn(LightField, err, device, nil)
	}
	_ = updatePointFn(LightField, float64(light), device, nil)
	return nil
}

func DecodeDropletTHLM(
	data string,
	dataBytes []byte,
	devDesc *codec.LoRaDeviceDescription,
	device *model.Device,
	updatePointFn codec.UpdateDevicePointFunc,
	updatePointErrFn codec.UpdateDevicePointErrorFunc,
	updateDeviceMetaTagsFn codec.UpdateDeviceMetaTagsFunc,
) error {
	err := DecodeDropletTHL(
		data,
		dataBytes,
		devDesc,
		device,
		updatePointFn,
		updatePointErrFn,
		updateDeviceMetaTagsFn,
	)
	if err != nil {
		return err
	}
	motion, err := dropletMotion(data)
	if err != nil {
		return updatePointErrFn(MotionField, err, device, nil)
	}
	_ = updatePointFn(MotionField, utils.BoolToFloat(motion), device, nil)
	return nil
}

func dropletTemp(data string) (float64, error) {
	if len(data) < 12 {
		return 0, fmt.Errorf("data too short for temperature: required=12 actual=%d", len(data))
	}
	v, err := strconv.ParseInt(data[10:12]+data[8:10], 16, 0)
	if err != nil {
		return 0, err
	}
	v_ := float64(int16(v)) / 100
	return v_, nil
}

func dropletPressure(data string) (float64, error) {
	if len(data) < 16 {
		return 0, fmt.Errorf("data too short for pressure: required=16 actual=%d", len(data))
	}
	v, err := strconv.ParseInt(data[14:16]+data[12:14], 16, 0)
	if err != nil {
		return 0, err
	}
	v_ := float64(v) / 10
	return v_, err
}

func dropletHumidity(data string) (int, error) {
	if len(data) < 18 {
		return 0, fmt.Errorf("data too short for humidity: required=18 actual=%d", len(data))
	}
	v, err := strconv.ParseInt(data[16:18], 16, 0)
	if err != nil {
		return 0, err
	}
	v = v & 127
	return int(v), nil
}

func dropletVoltage(data string) (float64, error) {
	if len(data) < 24 {
		return 0, fmt.Errorf("data too short for voltage: required=24 actual=%d", len(data))
	}
	v, err := strconv.ParseInt(data[22:24], 16, 0)
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
	if len(data) < 22 {
		return 0, fmt.Errorf("data too short for light: required=22 actual=%d", len(data))
	}
	v, err := strconv.ParseInt(data[20:22]+data[18:20], 16, 0)
	if err != nil {
		return 0, err
	}
	return int(v), nil
}

func dropletMotion(data string) (bool, error) {
	if len(data) < 18 {
		return false, fmt.Errorf("data too short for motion: required=18 actual=%d", len(data))
	}
	v, err := strconv.ParseInt(data[16:18], 16, 0)
	if err != nil {
		return false, err
	}
	return v > 127, nil
}
