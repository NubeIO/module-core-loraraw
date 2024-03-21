package pkg

import (
	"errors"
	"fmt"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/NubeIO/lib-module-go/nmodule"
	"github.com/NubeIO/lib-utils-go/boolean"
	"github.com/NubeIO/lib-utils-go/integer"
	"github.com/NubeIO/module-core-loraraw/decoder"
	"github.com/NubeIO/module-core-loraraw/utils"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/datatype"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/dto"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/nargs"
	log "github.com/sirupsen/logrus"
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

func (m *Module) networkUpdateSuccess(uuid string) error {
	var network model.Network
	network.InFault = false
	network.MessageLevel = dto.MessageLevel.Info
	network.MessageCode = dto.CommonFaultCode.Ok
	network.Message = ""
	network.LastOk = time.Now().UTC()
	err := m.grpcMarshaller.UpdateNetworkErrors(uuid, &network)
	if err != nil {
		log.Errorf("UpdateNetworkErrors() err: %s", err)
	}
	return err
}

func (m *Module) networkUpdateErr(uuid, port string, e error) error {
	var network model.Network
	network.InFault = true
	network.MessageLevel = dto.MessageLevel.Fail
	network.MessageCode = dto.CommonFaultCode.NetworkError
	network.Message = fmt.Sprintf("port: %s, message: %s", port, e.Error())
	network.LastFail = time.Now().UTC()
	err := m.grpcMarshaller.UpdateNetworkErrors(uuid, &network)
	if err != nil {
		log.Errorf("UpdateNetworkErrors() err: %s", err)
	}
	return err
}

func (m *Module) deviceUpdateSuccess(uuid string) error {
	var device model.Device
	device.InFault = false
	device.MessageLevel = dto.MessageLevel.Info
	device.MessageCode = dto.CommonFaultCode.Ok
	device.Message = ""
	device.LastOk = time.Now().UTC()
	err := m.grpcMarshaller.UpdateDeviceErrors(uuid, &device)
	if err != nil {
		log.Error(err)
	}
	return err
}

func (m *Module) deviceUpdateErr(uuid string, err error) error {
	var device model.Device
	device.InFault = true
	device.MessageLevel = dto.MessageLevel.Fail
	device.MessageCode = dto.CommonFaultCode.DeviceError
	device.Message = fmt.Sprintf("Error: %s", err.Error())
	device.LastFail = time.Now().UTC()
	err = m.grpcMarshaller.UpdateDeviceErrors(uuid, &device)
	if err != nil {
		log.Error(err)
	}
	return err
}

func (m *Module) pointUpdateSuccess(point *model.Point) error {
	if point == nil {
		return errors.New("lora-plugin: nil point to pointUpdateSuccess()")
	}
	point.InFault = false
	point.MessageLevel = dto.MessageLevel.Info
	point.MessageCode = dto.CommonFaultCode.Ok
	point.Message = ""
	point.LastOk = time.Now().UTC()
	err := m.grpcMarshaller.UpdatePointSuccess(point.UUID, point)
	if err != nil {
		log.Error(err)
	}
	return err
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
	commonData, fullData := decoder.DecodePayload(data, devDesc)
	if commonData == nil {
		return
	}
	log.Infof("sensor found. ID: %s, RSSI: %d, Type: %s", commonData.ID, commonData.Rssi, commonData.Sensor)
	_ = m.deviceUpdateSuccess(device.UUID)
	if fullData != nil {
		m.updateDevicePointValues(commonData, fullData, device)
	}
}

func (m *Module) getDeviceByLoRaAddress(address string) *model.Device {
	device, err := m.grpcMarshaller.GetOneDeviceByArgs(&nmodule.Opts{Args: &nargs.Args{AddressUUID: &address}})
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

// TODO: update to make more efficient for updating just the value (incl fault etc.)
func (m *Module) updatePointValue(body *model.Point, value float64, device *model.Device) error {
	// TODO: fix this so don't need to request the point for the UUID before hand
	pnt, err := m.grpcMarshaller.GetOnePointByArgs(&nmodule.Opts{Args: &nargs.Args{AddressUUID: body.AddressUUID, IoNumber: &body.IoNumber}})
	if err != nil {
		log.Errorf("issue on failed to find point: %v address_uuid: %s IO-ID: %s", err, *body.AddressUUID, body.IoNumber)
		return err
	}
	if pnt.IoType != "" && pnt.IoType != string(datatype.IOTypeRAW) {
		value = decoder.MicroEdgePointType(pnt.IoType, value, device.Model)
	}
	pointWriter := dto.PointWriter{
		OriginalValue: &value,
	}
	pwr, err := m.grpcMarshaller.PointWrite(pnt.UUID, &pointWriter) // TODO: look on it, faults messages were cleared out
	if err != nil {
		log.Error(err)
		return err
	}
	err = m.pointUpdateSuccess(&pwr.Point)
	return err
}

// updateDevicePointValues update all points under a device within commonSensorData and sensorStruct
func (m *Module) updateDevicePointValues(commonValues *decoder.CommonValues, sensorStruct interface{}, device *model.Device) {
	// manually update rssi + any other CommonValues
	pnt := new(model.Point)
	pnt.AddressUUID = &commonValues.ID
	pnt.IoNumber = utils.GetStructFieldJSONNameByName(sensorStruct, "Rssi")
	err := m.updatePointValue(pnt, float64(commonValues.Rssi), device)
	if err != nil {
		return
	}
	pnt.IoNumber = utils.GetStructFieldJSONNameByName(sensorStruct, "Snr")
	err = m.updatePointValue(pnt, float64(commonValues.Snr), device)
	if err != nil {
		return
	}
	// update all other fields in sensorStruct
	m.updateDevicePointValuesStruct(commonValues.ID, sensorStruct, "", device)
}

func (m *Module) updateDevicePointValuesStruct(deviceID string, sensorStruct interface{}, postfix string, device *model.Device) {
	pnt := new(model.Point)
	pnt.AddressUUID = &deviceID
	sensorRefl := reflect.ValueOf(sensorStruct)

	for i := 0; i < sensorRefl.NumField(); i++ {
		value := 0.0
		pnt.IoNumber = fmt.Sprintf("%s%s", utils.GetReflectFieldJSONName(sensorRefl.Type().Field(i)), postfix)
		field := sensorRefl.Field(i)

		switch field.Kind() {
		case reflect.Float32, reflect.Float64:
			value = field.Float()
		case reflect.Bool:
			value = utils.BoolToFloat(field.Bool())
		case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
			value = float64(field.Int())
		case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
			value = float64(field.Uint())
		case reflect.Struct:
			if _, ok := field.Interface().(decoder.CommonValues); !ok {
				m.updateDevicePointValuesStruct(deviceID, field.Interface(), postfix, device)
			}
			continue
		case reflect.Array:
			fallthrough
		case reflect.Slice:
			for j := 0; j < field.Len(); j++ {
				pf := fmt.Sprintf("%s_%d", postfix, j+1)
				v := field.Index(j).Interface()
				m.updateDevicePointValuesStruct(deviceID, v, pf, device)
			}
			continue
		default:
			continue
		}

		err := m.updatePointValue(pnt, value, device)
		if err != nil {
			return
		}
	}
}

func (m *Module) updatePluginMessage(messageLevel, message string) error {
	var plugin model.Plugin
	plugin.MessageLevel = messageLevel
	plugin.Message = message
	err := m.grpcMarshaller.UpdatePluginMessage(pluginName, &plugin)
	if err != nil {
		log.Errorf("updatePluginMessage() err: %s", err)
	}
	return err
}
