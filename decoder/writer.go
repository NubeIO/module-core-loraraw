package decoder

import (
	"github.com/NubeIO/lib-utils-go/boolean"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/datatype"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/dto"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
	log "github.com/sirupsen/logrus"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"strings"
)

func updateDeviceFault(sensor, deviceUUID string) {
	log.Infof("sensor found. Type: %s", sensor)
	_ = grpcMarshaller.UpdateDeviceFault(deviceUUID, &model.CommonFault{
		InFault: false,
		Message: "",
	})
}

func UpdateDevicePoint(name string, value float64, device *model.Device) error {
	pnt := selectPointByIoNumber(name, device)
	if pnt == nil {
		log.Debugf("failed to find point with address_uuid: %s and io_number: %s", *device.AddressUUID, name)
		newPoint, err := addPointFromName(device, name)
		if err != nil {
			log.Errorf("failed to create point with address_uuid: %s and io_number: %s", *device.AddressUUID, name)
			return err
		}
		pnt = newPoint
	}
	err := updatePointValue(pnt, value, device.Model)
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

func addPointFromName(deviceBody *model.Device, name string) (*model.Point, error) {
	point := new(model.Point)
	SetNewPointFields(deviceBody, point, name)
	point.EnableWriteable = boolean.NewFalse()
	pnt, err := savePoint(point)
	return pnt, err
}

func savePoint(point *model.Point) (*model.Point, error) {
	point.EnableWriteable = boolean.NewFalse()
	pnt, err := addPoint(point)
	return pnt, err
}

func addPoint(body *model.Point) (point *model.Point, err error) {
	body.ObjectType = "analog_input"
	body.IoType = string(datatype.IOTypeRAW)
	body.Name = strings.ToLower(body.Name)
	body.EnableWriteable = boolean.NewFalse()
	point, err = grpcMarshaller.CreatePoint(body) // TODO: in older one after creating there is an update operation
	if err != nil {
		return nil, err
	}
	return point, nil
}

func SetNewPointFields(deviceBody *model.Device, pointBody *model.Point, name string) {
	pointBody.Enable = boolean.NewTrue()
	pointBody.DeviceUUID = deviceBody.UUID
	pointBody.AddressUUID = deviceBody.AddressUUID
	pointBody.IsOutput = boolean.NewFalse()
	pointBody.Name = cases.Title(language.English).String(name)
	pointBody.IoNumber = name
	pointBody.ThingType = "point"
	pointBody.WriteMode = datatype.ReadOnly
}

func updatePointValue(pnt *model.Point, value float64, deviceModel string) error {
	if pnt.IoType != "" && pnt.IoType != string(datatype.IOTypeRAW) {
		value = MicroEdgePointType(pnt.IoType, value, deviceModel)
	}
	priority := map[string]*float64{"_16": &value}
	pointWriter := dto.PointWriter{
		OriginalValue: &value,
		Priority:      &priority,
	}
	_, err := grpcMarshaller.PointWrite(pnt.UUID, &pointWriter)
	if err != nil {
		log.Error(err)
		return err
	}
	return err
}
