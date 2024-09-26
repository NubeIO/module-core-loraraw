package pkg

import (
	"github.com/NubeIO/lib-utils-go/boolean"
	"github.com/NubeIO/module-core-loraraw/endec"
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

func (m *Module) updateDevicePoint(name string, value float64, device *model.Device) error {
	pnt := selectPointByIoNumber(name, device)
	if pnt == nil {
		log.Debugf("failed to find point with address_uuid: %s and io_number: %s", *device.AddressUUID, name)
		newPoint, err := m.addPointFromName(device, name)
		if err != nil {
			log.Errorf("failed to create point with address_uuid: %s and io_number: %s", *device.AddressUUID, name)
			return err
		}
		pnt = newPoint
	}
	err := m.updatePointValue(pnt, value, device.Model)
	if err != nil {
		return err
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

func (m *Module) addPointFromName(deviceBody *model.Device, name string) (*model.Point, error) {
	point := new(model.Point)
	setNewPointFields(deviceBody, point, name)
	point.EnableWriteable = boolean.NewFalse()
	pnt, err := m.savePoint(point)
	return pnt, err
}

func (m *Module) savePoint(point *model.Point) (*model.Point, error) {
	point.EnableWriteable = boolean.NewFalse()
	pnt, err := m.addPoint(point)
	return pnt, err
}

func setNewPointFields(deviceBody *model.Device, pointBody *model.Point, name string) {
	pointBody.Enable = boolean.NewTrue()
	pointBody.DeviceUUID = deviceBody.UUID
	pointBody.AddressUUID = deviceBody.AddressUUID
	pointBody.IsOutput = boolean.NewFalse()
	pointBody.Name = cases.Title(language.English).String(name)
	pointBody.IoNumber = name
	pointBody.ThingType = "point"
	pointBody.WriteMode = datatype.ReadOnly
}

func (m *Module) updatePointValue(pnt *model.Point, value float64, deviceModel string) error {
	if pnt.IoType != "" && pnt.IoType != string(datatype.IOTypeRAW) {
		value = endec.MicroEdgePointType(pnt.IoType, value, deviceModel)
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

func (m *Module) updateDeviceMetaTags(uuid string, metaTags []*model.DeviceMetaTag) error {
	err := m.grpcMarshaller.UpsertDeviceMetaTags(uuid, metaTags, nil)
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}
