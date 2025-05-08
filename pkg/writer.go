package pkg

import (
	"github.com/NubeIO/lib-utils-go/boolean"
	"github.com/NubeIO/module-core-loraraw/codecs/legacyDecoders"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/datatype"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/dto"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
	log "github.com/sirupsen/logrus"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func (m *Module) updateDeviceFault(sensor, deviceUUID string) {
	log.Infof("sensor found. Type: %s", sensor)
	_ = m.grpcMarshaller.UpdateDeviceFault(deviceUUID, &model.CommonFault{
		InFault: false,
		Message: "",
	})
}

func (m *Module) updateDevicePointSuccess(pointIDStr string, value float64, device *model.Device) error {
	return m.updateDevicePoint(pointIDStr, value, nil, device)
}

func (m *Module) updateDevicePointError(pointIDStr string, err error, device *model.Device) error {
	return m.updateDevicePoint(pointIDStr, 0, err, device)
}

func (m *Module) updateDevicePoint(pointIDStr string, value float64, err error, device *model.Device) error {
	pnt := selectPointByIoNumber(pointIDStr, device)
	if pnt == nil {
		log.Debugf("failed to find point with address_uuid: %s and io_number: %s", *device.AddressUUID, pointIDStr)
		newPoint, err := m.addPointFromName(device, pointIDStr)
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
	point := m.pointWriteQueue.DequeueUsingMessageId(messageId)
	if err != nil {
		_, _ = m.updateWrittenPointError(point, err)
	}
	_, _ = m.updateWrittenPointSuccess(point)
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

func (m *Module) addPointFromName(deviceBody *model.Device, pointIDStr string) (*model.Point, error) {
	point := new(model.Point)
	setNewPointFields(deviceBody, point, pointIDStr)
	point.EnableWriteable = boolean.NewFalse()
	pnt, err := m.savePoint(point)
	return pnt, err
}

func (m *Module) savePoint(point *model.Point) (*model.Point, error) {
	point.EnableWriteable = boolean.NewFalse()
	pnt, err := m.addPoint(point)
	return pnt, err
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
		log.Errorf("internalPointUpdate() error: %s", err)
		return nil, err
	}
	return &pwResponse.Point, nil
}

func (m *Module) updateWrittenPointError(point *model.Point, err error) (*model.Point, error) {
	pointWriter := &dto.PointWriter{
		Message:   err.Error(),
		Fault:     true,
		PollState: datatype.PointStateWriteOk,
	}
	pwResponse, err := m.grpcMarshaller.PointWrite(point.UUID, pointWriter)
	if err != nil {
		log.Errorf("internalPointUpdate() error: %s", err)
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
