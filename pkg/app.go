package pkg

import (
	"errors"
	"fmt"
	"github.com/NubeIO/lib-module-go/nmodule"
	"github.com/NubeIO/lib-utils-go/boolean"
	"github.com/NubeIO/lib-utils-go/integer"
	"github.com/NubeIO/module-core-loraraw/decoder"
	"github.com/NubeIO/module-core-loraraw/utils"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/datatype"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/nargs"
	log "github.com/sirupsen/logrus"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"reflect"
	"strings"
	"sync"
)

func (m *Module) addNetwork(body *model.Network) (network *model.Network, err error) {
	nets, err := m.grpcMarshaller.GetNetworksByPluginName(body.PluginName)
	if err != nil {
		return nil, err
	}
	for _, net := range nets {
		if net != nil {
			errMsg := fmt.Sprintf("only max one network is allowed with %s", m.moduleName)
			log.Errorf(errMsg)
			return nil, errors.New(errMsg)
		}
	}
	body.TransportType = "serial"
	if integer.IsUnitNil(body.SerialBaudRate) {
		body.SerialBaudRate = integer.NewUint(38400)
	}
	network, err = m.grpcMarshaller.CreateNetwork(body)
	if err != nil {
		return nil, err
	}
	m.networkUUID = network.UUID
	go m.run()
	return network, nil
}

func (m *Module) addDevice(body *model.Device, withPoints bool) (device *model.Device, err error) {
	*body.AddressUUID = strings.ToUpper(*body.AddressUUID)
	device, _ = m.grpcMarshaller.GetOneDeviceByArgs(&nmodule.Opts{Args: &nargs.Args{AddressUUID: body.AddressUUID}})
	if device != nil {
		errMsg := fmt.Sprintf("the lora ID (address_uuid) must be unique: %s", *body.AddressUUID)
		log.Errorf(errMsg)
		return nil, errors.New(errMsg)
	}
	device, err = m.grpcMarshaller.CreateDevice(body)
	if err != nil {
		return nil, err
	}
	if withPoints {
		err = m.addDevicePoints(device)
		if err != nil {
			_ = m.grpcMarshaller.DeleteDevice(device.UUID)
			return nil, err
		}
	}
	return device, nil
}

func (m *Module) addPoint(body *model.Point) (point *model.Point, err error) {
	body.ObjectType = "analog_input"
	body.IoType = string(datatype.IOTypeRAW)
	body.Name = strings.ToLower(body.Name)
	body.EnableWriteable = boolean.NewFalse()
	point, err = m.grpcMarshaller.CreatePoint(body) // TODO: in older one after creating there is an update operation
	if err != nil {
		return nil, err
	}
	return point, nil
}

func (m *Module) deletePoint(body *model.Point) (success bool, err error) {
	// TODO: For now this db call has been removed, so that point deletes of lora points is not allowed by the user; can only be deleted by the whole device.
	/*
		success, err = m.db.DeletePoint(body.UUID)
		if err != nil {
			return false, err
		}
	*/
	return success, nil
}

func (m *Module) handleSerialPayload(data string) {
	if m.networkUUID == "" {
		return
	}
	if !decoder.ValidPayload(data) {
		return
	}
	log.Debugf("uplink: %s", data)
	device := m.getDeviceByLoRaAddress(decoder.DecodeAddress(data))
	if device == nil {
		id := decoder.DecodeAddress(data) // show user messages from lora
		rssi := decoder.DecodeRSSI(data)
		log.Infof("message from non-added sensor. ID: %s, RSSI: %d", id, rssi)
		return
	}
	devDesc := decoder.GetDeviceDescription(device)
	if devDesc == &decoder.NilLoRaDeviceDescription {
		return
	}
	err := decoder.DecodePayload(data, devDesc, device)
	if err != nil {
		log.Errorf(err.Error())
	}
}

func (m *Module) getDeviceByLoRaAddress(address string) *model.Device {
	opts := &nmodule.Opts{Args: &nargs.Args{AddressUUID: &address, WithPoints: true}}
	device, err := m.grpcMarshaller.GetOneDeviceByArgs(opts)
	if err != nil {
		return nil
	}
	return device
}

// TODO: need better way to add/update CommonValues points instead of adding/updating the rssi point manually in each func
// addDevicePoints add all points related to a device
func (m *Module) addDevicePoints(deviceBody *model.Device) error {
	network, err := m.grpcMarshaller.GetNetwork(deviceBody.NetworkUUID)
	if err != nil {
		log.Errorf("addDevicePoints() err: %s", err)
		return err
	}
	if network.PluginName != m.moduleName {
		errMsg := fmt.Sprintf("incorrect network plugin type, must be %s", m.moduleName)
		log.Errorln(errMsg)
		return errors.New(errMsg)
	}

	points := decoder.GetDevicePointsStruct(deviceBody)
	// TODO: should check this before the device is even added in the wizard
	if points == struct{}{} {
		log.Errorf("addDevicePoints() incorrect device model, try THLM %s", err)
		return errors.New("addDevicePoints() no device description or points found for this device")
	}
	pointsRefl := reflect.ValueOf(points)
	m.addPointsFromName(deviceBody, "Rssi", "Snr")
	m.addPointsFromStruct(deviceBody, pointsRefl, "")
	return nil
}

func (m *Module) addPointFromName(deviceBody *model.Device, name string) (*model.Point, error) {
	point := new(model.Point)
	m.setNewPointFields(deviceBody, point, name)
	point.EnableWriteable = boolean.NewFalse()
	pnt, err := m.savePoint(point)
	return pnt, err
}

func (m *Module) savePoint(point *model.Point) (*model.Point, error) {
	point.EnableWriteable = boolean.NewFalse()
	pnt, err := m.addPoint(point)
	return pnt, err
}

func (m *Module) addPointsFromName(deviceBody *model.Device, names ...string) {
	var points []*model.Point
	for _, name := range names {
		pointName := utils.GetStructFieldJSONNameByName(decoder.CommonValues{}, name)
		point := new(model.Point)
		m.setNewPointFields(deviceBody, point, pointName)
		point.EnableWriteable = boolean.NewFalse()
		points = append(points, point)
	}
	m.savePoints(points)
}

func (m *Module) addPointsFromStruct(deviceBody *model.Device, pointsRefl reflect.Value, postfix string) {
	var points []*model.Point
	for i := 0; i < pointsRefl.NumField(); i++ {
		field := pointsRefl.Field(i)
		if field.Kind() == reflect.Struct {
			if _, ok := field.Interface().(decoder.CommonValues); !ok {
				m.addPointsFromStruct(deviceBody, pointsRefl.Field(i), postfix)
			}
			continue
		} else if field.Kind() == reflect.Array || field.Kind() == reflect.Slice {
			for j := 0; j < field.Len(); j++ {
				pf := fmt.Sprintf("%s_%d", postfix, j+1)
				v := field.Index(j)
				m.addPointsFromStruct(deviceBody, v, pf)
			}
			continue
		}
		pointName := utils.GetReflectFieldJSONName(pointsRefl.Type().Field(i))
		if postfix != "" {
			pointName = fmt.Sprintf("%s%s", pointName, postfix)
		}
		point := new(model.Point)
		m.setNewPointFields(deviceBody, point, pointName)
		point.EnableWriteable = boolean.NewFalse()
		points = append(points, point)
	}
	m.savePoints(points)
}

func (m *Module) savePoints(points []*model.Point) {
	var wg sync.WaitGroup
	for _, point := range points {
		wg.Add(1)
		point := point
		go func() {
			defer wg.Done()
			point.EnableWriteable = boolean.NewFalse()
			_, err := m.addPoint(point)
			if err != nil {
				log.Errorf("issue on addPoint: %s", err)
			}
		}()
	}
	wg.Wait()
}

func (m *Module) setNewPointFields(deviceBody *model.Device, pointBody *model.Point, name string) {
	pointBody.Enable = boolean.NewTrue()
	pointBody.DeviceUUID = deviceBody.UUID
	pointBody.AddressUUID = deviceBody.AddressUUID
	pointBody.IsOutput = boolean.NewFalse()
	pointBody.Name = cases.Title(language.English).String(name)
	pointBody.IoNumber = name
	pointBody.ThingType = "point"
	pointBody.WriteMode = datatype.ReadOnly
}

// updateDevicePointsAddress by its lora id and type as in temp or lux
func (m *Module) updateDevicePointsAddress(body *model.Device) error {
	dev, err := m.grpcMarshaller.GetDevice(body.UUID, &nmodule.Opts{Args: &nargs.Args{WithPoints: true}})
	if err != nil {
		return err
	}
	for _, pt := range dev.Points {
		pt.AddressUUID = body.AddressUUID
		pt.EnableWriteable = boolean.NewFalse()
		_, err = m.grpcMarshaller.UpdatePoint(pt.UUID, pt)
		if err != nil {
			log.Errorf("issue on UpdatePoint updateDevicePointsAddress(): %s", err)
			return err
		}
	}
	return nil
}

func (m *Module) updatePluginMessage(messageLevel, message string) error {
	err := m.grpcMarshaller.UpdatePluginMessage(m.moduleName, &model.Plugin{
		MessageLevel: messageLevel,
		Message:      message,
	})
	if err != nil {
		log.Errorf("updatePluginMessage() err: %s", err)
	}
	return err
}

func (m *Module) selectPointByIoNumber(ioNumber string, device *model.Device) *model.Point {
	for _, pnt := range device.Points {
		if pnt.IoNumber == ioNumber {
			return pnt
		}
	}
	return nil
}
