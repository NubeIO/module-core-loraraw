package decoder

import (
	"errors"
	"github.com/NubeIO/module-core-loraraw/schema"
	"github.com/NubeIO/nubeio-rubix-lib-helpers-go/pkg/nube/thermistor"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/datatype"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
	"strconv"
)

type TMicroEdge struct {
	CommonValues
	Voltage float64 `json:"voltage"`
	Pulse   int     `json:"pulse"`
	AI1     float64 `json:"ai_1"`
	AI2     float64 `json:"ai_2"`
	AI3     float64 `json:"ai_3"`
}

func GetPointsStructME() interface{} {
	return TMicroEdge{}
}

func CheckPayloadLengthME(data string) bool {
	dl := len(data)
	return dl == 36 || dl == 32 || dl == 44
}

func DecodeME(data string, devDesc *LoRaDeviceDescription, device *model.Device) error {
	commonValues := &CommonValues{}
	decodeCommonValues(commonValues, data, devDesc.Model)
	if commonValues == nil {
		return errors.New("invalid common values")
	}

	updateDeviceFault(commonValues.ID, commonValues.Sensor, device.UUID, commonValues.Rssi)

	err := updateDevicePoint("rssi", float64(commonValues.Rssi), device)
	if err != nil {
		return err
	}

	err = updateDevicePoint("snr", float64(commonValues.Snr), device)
	if err != nil {
		return err
	}

	p := pulse(data)
	a1 := ai1(data)
	a2 := ai2(data)
	a3 := ai3(data)
	vol := voltage(data)

	_ = updateDevicePoint("pulse", float64(p), device)
	_ = updateDevicePoint("ai_1", a1, device)
	_ = updateDevicePoint("ai_2", a2, device)
	_ = updateDevicePoint("ai_3", a3, device)
	_ = updateDevicePoint("voltage", vol, device)

	return nil
}

func pulse(data string) int {
	v, _ := strconv.ParseInt(data[8:16], 16, 0)
	return int(v)
}

func ai1(data string) float64 {
	v, _ := strconv.ParseInt(data[18:22], 16, 0)
	return float64(v)
}

func ai2(data string) float64 {
	v, _ := strconv.ParseInt(data[22:26], 16, 0)
	return float64(v)
}

func ai3(data string) float64 {
	v, _ := strconv.ParseInt(data[26:30], 16, 0)
	return float64(v)
}

func voltage(data string) float64 {
	v, _ := strconv.ParseInt(data[16:18], 16, 0)
	v_ := float64(v) / 50
	return v_
}

func MicroEdgePointType(pointType string, value float64, deviceModel string) float64 {
	switch datatype.IOType(pointType) {
	case datatype.IOTypeRAW:
		return value
	case datatype.IOTypeDigital:
		if value >= 1000 {
			return 0
		} else {
			return 1
		}
	case datatype.IOTypeThermistor10K:
		var r float64
		if deviceModel == schema.DeviceModelMicroEdgeV2 {
			v := value / 1023 * 3.29
			r = (10000 * v) / (3.29 - v)
		} else {
			r = ((16620 * value) - (1023 * 3300)) / (1023 - value)
		}
		f, _ := thermistor.ResistanceToTemperature(r, thermistor.T210K)
		return f
	case datatype.IOTypeVoltageDC:
		output := (value / 1024) * 10
		return output
	default:
		return value
	}
}
