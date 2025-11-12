package pkg

import (
	"strings"

	"github.com/NubeIO/lib-utils-go/boolean"
	"github.com/NubeIO/lib-utils-go/float"
	"github.com/NubeIO/lib-utils-go/integer"
	"github.com/NubeIO/module-core-loraraw/codec"
	"github.com/NubeIO/module-core-loraraw/codecs/legacyDecoders"
	"github.com/NubeIO/module-core-loraraw/codecs/rubixDataEncoding"
	"github.com/NubeIO/module-core-loraraw/schema"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/datatype"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/dto"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
	log "github.com/sirupsen/logrus"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type uartPointConfig struct {
	Name                string
	DataType            string
	WriteMode           datatype.WriteMode
	HistoryEnable       *bool
	HistoryType         datatype.HistoryType
	HistoryInterval     *int
	HistoryCOVThreshold *float64
}

func getUARTPointConfig(pointID string) *uartPointConfig {
	configs := map[string]*uartPointConfig{
		"1":  {"Communication Status", "uint8", datatype.ReadOnly, boolean.NewTrue(), datatype.HistoryTypeCovAndInterval, integer.New(60), float.New(0.01)},
		"2":  {"Unit Type", "uint8", datatype.ReadOnly, boolean.NewFalse(), datatype.HistoryTypeInterval, integer.New(15), float.New(0.01)},
		"3":  {"Has Economy", "bool", datatype.ReadOnly, boolean.NewFalse(), datatype.HistoryTypeInterval, integer.New(15), float.New(0.01)},
		"10": {"Has Mode Cool", "bool", datatype.ReadOnly, boolean.NewFalse(), datatype.HistoryTypeInterval, integer.New(15), float.New(0.01)},
		"11": {"Has Mode Dry", "bool", datatype.ReadOnly, boolean.NewFalse(), datatype.HistoryTypeInterval, integer.New(15), float.New(0.01)},
		"12": {"Has Mode Fan", "bool", datatype.ReadOnly, boolean.NewFalse(), datatype.HistoryTypeInterval, integer.New(15), float.New(0.01)},
		"13": {"Has Mode Heat", "bool", datatype.ReadOnly, boolean.NewFalse(), datatype.HistoryTypeInterval, integer.New(15), float.New(0.01)},
		"14": {"Has Mode Auto", "bool", datatype.ReadOnly, boolean.NewFalse(), datatype.HistoryTypeInterval, integer.New(15), float.New(0.01)},
		"15": {"Has Fan Auto", "bool", datatype.ReadOnly, boolean.NewFalse(), datatype.HistoryTypeInterval, integer.New(15), float.New(0.01)},
		"16": {"Has Fan High", "bool", datatype.ReadOnly, boolean.NewFalse(), datatype.HistoryTypeInterval, integer.New(15), float.New(0.01)},
		"17": {"Has Fan Medium", "bool", datatype.ReadOnly, boolean.NewFalse(), datatype.HistoryTypeInterval, integer.New(15), float.New(0.01)},
		"18": {"Has Fan Low", "bool", datatype.ReadOnly, boolean.NewFalse(), datatype.HistoryTypeInterval, integer.New(15), float.New(0.01)},
		"19": {"Has Fan Quiet", "bool", datatype.ReadOnly, boolean.NewFalse(), datatype.HistoryTypeInterval, integer.New(15), float.New(0.01)},
		"20": {"Vertical Louver Step Count", "uint8", datatype.ReadOnly, boolean.NewFalse(), datatype.HistoryTypeInterval, integer.New(15), float.New(0.01)},
		"21": {"Has Vertical Louver Swing", "bool", datatype.ReadOnly, boolean.NewFalse(), datatype.HistoryTypeInterval, integer.New(15), float.New(0.01)},
		"22": {"Vertical Louver 1 Step Count", "uint8", datatype.ReadOnly, boolean.NewFalse(), datatype.HistoryTypeInterval, integer.New(15), float.New(0.01)},
		"23": {"Has Vertical Louver 1 Swing", "bool", datatype.ReadOnly, boolean.NewFalse(), datatype.HistoryTypeInterval, integer.New(15), float.New(0.01)},
		"24": {"Vertical Louver 2 Step Count", "uint8", datatype.ReadOnly, boolean.NewFalse(), datatype.HistoryTypeInterval, integer.New(15), float.New(0.01)},
		"25": {"Has Vertical Louver 2 Swing", "bool", datatype.ReadOnly, boolean.NewFalse(), datatype.HistoryTypeInterval, integer.New(15), float.New(0.01)},
		"26": {"Vertical Louver 3 Step Count", "uint8", datatype.ReadOnly, boolean.NewFalse(), datatype.HistoryTypeInterval, integer.New(15), float.New(0.01)},
		"27": {"Has Vertical Louver 3 Swing", "bool", datatype.ReadOnly, boolean.NewFalse(), datatype.HistoryTypeInterval, integer.New(15), float.New(0.01)},
		"28": {"Vertical Louver 4 Step Count", "uint8", datatype.ReadOnly, boolean.NewFalse(), datatype.HistoryTypeInterval, integer.New(15), float.New(0.01)},
		"29": {"Has Vertical Louver 4 Swing", "bool", datatype.ReadOnly, boolean.NewFalse(), datatype.HistoryTypeInterval, integer.New(15), float.New(0.01)},
		"30": {"Horizontal Louver Step Count", "uint8", datatype.ReadOnly, boolean.NewFalse(), datatype.HistoryTypeInterval, integer.New(15), float.New(0.01)},
		"31": {"Has Horizontal Louver Swing", "bool", datatype.ReadOnly, boolean.NewFalse(), datatype.HistoryTypeInterval, integer.New(15), float.New(0.01)},
		"32": {"Horizontal Louver 1 Step Count", "uint8", datatype.ReadOnly, boolean.NewFalse(), datatype.HistoryTypeInterval, integer.New(15), float.New(0.01)},
		"33": {"Has Horizontal Louver 1 Swing", "bool", datatype.ReadOnly, boolean.NewFalse(), datatype.HistoryTypeInterval, integer.New(15), float.New(0.01)},
		"34": {"Horizontal Louver 2 Step Count", "uint8", datatype.ReadOnly, boolean.NewFalse(), datatype.HistoryTypeInterval, integer.New(15), float.New(0.01)},
		"35": {"Has Horizontal Louver 2 Swing", "bool", datatype.ReadOnly, boolean.NewFalse(), datatype.HistoryTypeInterval, integer.New(15), float.New(0.01)},
		"36": {"Horizontal Louver 3 Step Count", "uint8", datatype.ReadOnly, boolean.NewFalse(), datatype.HistoryTypeInterval, integer.New(15), float.New(0.01)},
		"37": {"Has Horizontal Louver 3 Swing", "bool", datatype.ReadOnly, boolean.NewFalse(), datatype.HistoryTypeInterval, integer.New(15), float.New(0.01)},
		"38": {"Horizontal Louver 4 Step Count", "uint8", datatype.ReadOnly, boolean.NewFalse(), datatype.HistoryTypeInterval, integer.New(15), float.New(0.01)},
		"39": {"Has Horizontal Louver 4 Swing", "bool", datatype.ReadOnly, boolean.NewFalse(), datatype.HistoryTypeInterval, integer.New(15), float.New(0.01)},
		"40": {"Set Operation Status", "bool", datatype.WriteAlways, boolean.NewTrue(), datatype.HistoryTypeCovAndInterval, integer.New(15), float.New(0.01)},
		"41": {"Set Operation Mode", "uint8", datatype.WriteAlways, boolean.NewTrue(), datatype.HistoryTypeCovAndInterval, integer.New(15), float.New(0.01)},
		"42": {"Set Temperature", "temp", datatype.WriteAlways, boolean.NewTrue(), datatype.HistoryTypeCovAndInterval, integer.New(15), float.New(0.01)},
		"43": {"Set Fan", "uint8", datatype.WriteAlways, boolean.NewTrue(), datatype.HistoryTypeCovAndInterval, integer.New(15), float.New(0.01)},
		"44": {"Vertical Louver Current Position", "uint8", datatype.WriteAlways, boolean.NewTrue(), datatype.HistoryTypeCovAndInterval, integer.New(15), float.New(0.01)},
		"45": {"Vertical Louver Swing", "bool", datatype.WriteAlways, boolean.NewTrue(), datatype.HistoryTypeCovAndInterval, integer.New(15), float.New(0.01)},
		"46": {"Vertical Louver 1 Current Position", "uint8", datatype.WriteAlways, boolean.NewTrue(), datatype.HistoryTypeCovAndInterval, integer.New(15), float.New(0.01)},
		"47": {"Verticle Louver 1 Swing", "bool", datatype.WriteAlways, boolean.NewTrue(), datatype.HistoryTypeCovAndInterval, integer.New(15), float.New(0.01)},
		"48": {"Vertical Louver 2 Current Position", "uint8", datatype.WriteAlways, boolean.NewTrue(), datatype.HistoryTypeCovAndInterval, integer.New(15), float.New(0.01)},
		"49": {"Verticle Louver 2 Swing", "bool", datatype.WriteAlways, boolean.NewTrue(), datatype.HistoryTypeCovAndInterval, integer.New(15), float.New(0.01)},
		"50": {"Vertical Louver 3 Current Position", "uint8", datatype.WriteAlways, boolean.NewTrue(), datatype.HistoryTypeCovAndInterval, integer.New(15), float.New(0.01)},
		"51": {"Verticle Louver 3 Swing", "bool", datatype.WriteAlways, boolean.NewTrue(), datatype.HistoryTypeCovAndInterval, integer.New(15), float.New(0.01)},
		"52": {"Vertical Louver 4 Current Position", "uint8", datatype.WriteAlways, boolean.NewTrue(), datatype.HistoryTypeCovAndInterval, integer.New(15), float.New(0.01)},
		"53": {"Verticle Louver 4 Swing", "bool", datatype.WriteAlways, boolean.NewTrue(), datatype.HistoryTypeCovAndInterval, integer.New(15), float.New(0.01)},
		"54": {"Horizontal Louver Current Position", "uint8", datatype.WriteAlways, boolean.NewTrue(), datatype.HistoryTypeCovAndInterval, integer.New(15), float.New(0.01)},
		"55": {"Horizontal Louver Swing", "bool", datatype.WriteAlways, boolean.NewTrue(), datatype.HistoryTypeCovAndInterval, integer.New(15), float.New(0.01)},
		"56": {"Room Temperature", "temp", datatype.ReadOnly, boolean.NewTrue(), datatype.HistoryTypeInterval, integer.New(15), float.New(0.01)},
		"57": {"Error State", "uint16", datatype.ReadOnly, boolean.NewTrue(), datatype.HistoryTypeCovAndInterval, integer.New(60), float.New(0.01)},
		"58": {"Error Code", "uint16", datatype.ReadOnly, boolean.NewTrue(), datatype.HistoryTypeCovAndInterval, integer.New(60), float.New(0.01)},
		"59": {"Set Economy", "bool", datatype.WriteAlways, boolean.NewTrue(), datatype.HistoryTypeCovAndInterval, integer.New(15), float.New(0.01)},
	}

	return configs[pointID]
}

func (m *Module) updateDeviceFault(sensor, deviceUUID string) {
	log.Infof("sensor found. Type: %s", sensor)
	_ = m.grpcMarshaller.UpdateDeviceFault(deviceUUID, &model.CommonFault{
		InFault: false,
		Message: "",
	})
}

func (m *Module) updateDevicePointSuccess(pointIDStr string, value float64, device *model.Device, devDesc *codec.LoRaDeviceDescription) error {
	return m.updateDevicePoint(pointIDStr, value, nil, device, devDesc)
}

func (m *Module) updateDevicePointError(pointIDStr string, err error, device *model.Device, devDesc *codec.LoRaDeviceDescription) error {
	return m.updateDevicePoint(pointIDStr, 0, err, device, devDesc)
}

func (m *Module) updateDevicePoint(pointIDStr string, value float64, err error, device *model.Device, devDesc *codec.LoRaDeviceDescription) error {
	pnt := selectPointByIoNumber(pointIDStr, device)
	if pnt == nil {
		log.Debugf("failed to find point with address_uuid: %s and io_number: %s", *device.AddressUUID, pointIDStr)
		newPoint, err := m.addPointFromName(device, pointIDStr, devDesc)
		if err != nil {
			log.Errorf("failed to create point with address_uuid: %s and io_number: %s", *device.AddressUUID, pointIDStr)
			return err
		}
		pnt = newPoint
	}
	if err != nil {
		err = m.updatePointValueError(pnt, err)
	} else {
		err = m.updatePointValueSuccess(pnt, value, device.Model)
	}
	if err != nil {
		return err
	}
	return nil
}

func (m *Module) updateDeviceWrittenPointSuccess(pointIDStr string, value float64, messageId uint8, device *model.Device) error {
	return m.updateDeviceWrittenPoint(pointIDStr, value, nil, messageId, device)
}

func (m *Module) updateDeviceWrittenPointError(pointIDStr string, err error, messageId uint8, device *model.Device) error {
	return m.updateDeviceWrittenPoint(pointIDStr, 0, err, messageId, device)
}

func (m *Module) updateDeviceWrittenPoint(pointIDStr string, value float64, err error, messageId uint8, device *model.Device) error {
	point := m.pointWriteQueueManager.DequeueUsingMessageId(device.UUID, messageId)
	if point == nil {
		log.Errorf("failed to find point with messageId: %d", messageId)
		return nil
	}
	if err != nil {
		_, _ = m.updateWrittenPointError(point, err)
	} else {
		_, _ = m.updateWrittenPointSuccess(point)
	}
	return nil
}

func selectPointByIoNumber(ioNumber string, device *model.Device) *model.Point {
	for _, pnt := range device.Points {
		if pnt.IoNumber == ioNumber {
			return pnt
		}
	}
	return nil
}

func (m *Module) addPointFromName(deviceBody *model.Device, pointIDStr string, devDesc *codec.LoRaDeviceDescription) (*model.Point, error) {
	point := new(model.Point)
	if devDesc != nil && devDesc.Model == schema.DeviceModelUART {
		parts := strings.Split(pointIDStr, "-")
		if len(parts) == 2 {
			setNewPointFieldsUART(deviceBody, point, parts[1])
		} else {
			setNewPointFields(deviceBody, point, pointIDStr)
		}
	} else {
		setNewPointFields(deviceBody, point, pointIDStr)
	}
	point.EnableWriteable = boolean.NewFalse()
	pnt, err := m.savePoint(point)
	return pnt, err
}

func setNewPointFieldsUART(deviceBody *model.Device, pointBody *model.Point, pointIDStr string) {
	uartPointConfig := getUARTPointConfig(pointIDStr)

	pointBody.Name = uartPointConfig.Name
	pointBody.IoNumber = uartPointConfig.Name
	pointBody.DataType = uartPointConfig.DataType
	pointBody.WriteMode = uartPointConfig.WriteMode
	pointBody.Enable = uartPointConfig.HistoryEnable
	pointBody.HistoryType = uartPointConfig.HistoryType
	pointBody.HistoryInterval = uartPointConfig.HistoryInterval
	pointBody.HistoryCOVThreshold = uartPointConfig.HistoryCOVThreshold
	pointBody.DeviceUUID = deviceBody.UUID
	pointBody.AddressUUID = deviceBody.AddressUUID
	pointBody.IsOutput = boolean.NewFalse()
	pointBody.ThingType = "point"
}

func setNewPointFields(deviceBody *model.Device, pointBody *model.Point, pointIDStr string) {
	pointBody.Enable = boolean.NewTrue()
	pointBody.DeviceUUID = deviceBody.UUID
	pointBody.AddressUUID = deviceBody.AddressUUID
	pointBody.IsOutput = boolean.NewFalse()
	pointBody.Name = cases.Title(language.English).String(pointIDStr)
	pointBody.IoNumber = pointIDStr
	pointBody.ThingType = "point"
	pointBody.WriteMode = datatype.ReadOnly
	pointBody.DataType, _ = rubixDataEncoding.GetMetaDataKey(pointIDStr)
}

func (m *Module) savePoint(point *model.Point) (*model.Point, error) {
	point.EnableWriteable = boolean.NewFalse()
	pnt, err := m.addPoint(point)
	return pnt, err
}

func (m *Module) updatePointValueSuccess(pnt *model.Point, value float64, deviceModel string) error {
	if pnt.IoType != "" && pnt.IoType != string(datatype.IOTypeRAW) {
		value = legacyDecoders.MicroEdgePointType(pnt.IoType, value, deviceModel)
	}
	priority := map[string]*float64{"_16": &value}
	pointWriter := dto.PointWriter{
		OriginalValue: &value,
		Priority:      &priority,
	}
	_, err := m.grpcMarshaller.PointWrite(pnt.UUID, &pointWriter)
	if err != nil {
		log.Error(err)
		return err
	}
	return err
}

func (m *Module) updatePointValueError(pnt *model.Point, err error) error {
	pointWriter := dto.PointWriter{
		Message: err.Error(),
		Fault:   true,
	}
	_, err = m.grpcMarshaller.PointWrite(pnt.UUID, &pointWriter)
	if err != nil {
		log.Error(err)
		return err
	}
	return err
}

func (m *Module) updateWrittenPointSuccess(point *model.Point) (*model.Point, error) {
	pointWriter := &dto.PointWriter{
		OriginalValue: point.WriteValue,
		Message:       "",
		Fault:         false,
		PollState:     datatype.PointStateWriteOk,
	}
	pwResponse, err := m.grpcMarshaller.PointWrite(point.UUID, pointWriter)
	if err != nil {
		log.Errorf("updateWrittenPointSuccess() error: %s", err)
		return nil, err
	}
	return &pwResponse.Point, nil
}

func (m *Module) updateWrittenPointError(point *model.Point, err error) (*model.Point, error) {
	pointWriter := &dto.PointWriter{
		OriginalValue: point.OriginalValue, // skip point writes if both OriginalValue and Priority is nil
		Message:       err.Error(),
		Fault:         true,
		PollState:     datatype.PointStateApiWriteFailed,
	}
	pwResponse, err := m.grpcMarshaller.PointWrite(point.UUID, pointWriter)
	if err != nil {
		log.Errorf("updateWrittenPointError() error: %s", err)
		return nil, err
	}
	return &pwResponse.Point, nil
}

func (m *Module) updateDeviceMetaTags(uuid string, metaTags []*model.DeviceMetaTag) error {
	err := m.grpcMarshaller.UpsertDeviceMetaTags(uuid, metaTags, nil)
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}
