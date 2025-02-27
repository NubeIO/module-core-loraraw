package endec

import (
	"github.com/NubeIO/module-core-loraraw/schema"
	"github.com/NubeIO/nubeio-rubix-lib-helpers-go/pkg/nube/thermistor"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/datatype"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
	"strconv"
)

const (
	MEVoltageField = "voltage"
	PulseField     = "pulse"
	AI1Field       = "ai_1"
	AI2Field       = "ai_2"
	AI3Field       = "ai_3"
)

func GetMePointNames() []string {
	commonValueFields := GetCommonValueNames()
	tMicroEdgeFields := []string{
		MEVoltageField,
		PulseField,
		AI1Field,
		AI2Field,
		AI3Field,
	}
	return append(commonValueFields, tMicroEdgeFields...)
}

func CheckPayloadLengthME(data string) bool {
	dl := len(data)
	return dl == 36 || dl == 32 || dl == 44
}

func DecodeME(
	data string,
	_ *LoRaDeviceDescription,
	device *model.Device,
	updatePointFn UpdateDevicePointFunc,
	_ UpdateDeviceMetaTagsFunc,
	_ DequeuePointWriteFunc,
	_ InternalPointUpdate,
) error {
	p, err := pulse(data)
	if err != nil {
		return err
	}
	vol, err := voltage(data)
	if err != nil {
		return err
	}
	a1, err := ai1(data)
	if err != nil {
		return err
	}
	a2, err := ai2(data)
	if err != nil {
		return err
	}
	a3, err := ai3(data)
	if err != nil {
		return err
	}

	_ = updatePointFn(PulseField, float64(p), device)
	_ = updatePointFn(MEVoltageField, vol, device)
	_ = updatePointFn(AI1Field, a1, device)
	_ = updatePointFn(AI2Field, a2, device)
	_ = updatePointFn(AI3Field, a3, device)

	return nil
}

func pulse(data string) (int, error) {
	v, err := strconv.ParseInt(data[8:16], 16, 0)
	if err != nil {
		return 0, err
	}
	return int(v), nil
}

func ai1(data string) (float64, error) {
	v, err := strconv.ParseInt(data[18:22], 16, 0)
	if err != nil {
		return 0, err
	}
	return float64(v), nil
}

func ai2(data string) (float64, error) {
	v, err := strconv.ParseInt(data[22:26], 16, 0)
	if err != nil {
		return 0, err
	}
	return float64(v), nil
}

func ai3(data string) (float64, error) {
	v, err := strconv.ParseInt(data[26:30], 16, 0)
	if err != nil {
		return 0, err
	}
	return float64(v), nil
}

func voltage(data string) (float64, error) {
	v, err := strconv.ParseInt(data[16:18], 16, 0)
	if err != nil {
		return 0, err
	}
	v_ := float64(v) / 50
	return v_, nil
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
