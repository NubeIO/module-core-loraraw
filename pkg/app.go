package pkg

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/dto"
	"reflect"
	"strings"
	"sync"

	"github.com/NubeIO/module-core-loraraw/aesutils"

	"github.com/NubeIO/lib-module-go/nmodule"
	"github.com/NubeIO/lib-utils-go/boolean"
	"github.com/NubeIO/lib-utils-go/integer"
	"github.com/NubeIO/module-core-loraraw/endec"
	"github.com/NubeIO/module-core-loraraw/utils"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/datatype"
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
	if utils.IsWriteable(body.WriteMode) {
		dev, err := m.grpcMarshaller.GetDevice(body.DeviceUUID)
		if err != nil {
			return nil, err
		}
		body.AddressUUID = dev.AddressUUID
		body.EnableWriteable = boolean.NewTrue()
		body.WritePollRequired = boolean.NewTrue()
	} else {
		body = utils.ResetWriteableProperties(body)
	}
	point, err = m.grpcMarshaller.CreatePoint(body) // TODO: in older one after creating there is an update operation
	if err != nil {
		return nil, err
	}
	return point, nil
}

func (m *Module) updatePoint(uuid string, body *model.Point) (*model.Point, error) {
	if utils.IsWriteable(body.WriteMode) {
		dev, err := m.grpcMarshaller.GetDevice(body.DeviceUUID)
		if err != nil {
			return nil, err
		}
		body.AddressUUID = dev.AddressUUID
		body.EnableWriteable = boolean.NewTrue()
		body.WritePollRequired = boolean.NewTrue()
	} else {
		body = utils.ResetWriteableProperties(body)
	}
	pnt, err := m.grpcMarshaller.UpdatePoint(uuid, body)
	if err != nil {
		return nil, err
	}
	return pnt, nil
}

func (m *Module) deletePoint(_ *model.Point) (success bool, err error) {
	// TODO: For now this db call has been removed, so that point deletes of lora points is not allowed by the user; can only be deleted by the whole device.
	/*
		success, err = m.db.DeletePoint(body.UUID)
		if err != nil {
			return false, err
		}
	*/
	return success, nil
}

func (m *Module) writePoint(pointUUID string, body *dto.PointWriter) (*model.Point, error) {
	body.IgnorePresentValueUpdate = true
	body.PollState = datatype.PointStateApiUpdatePending
	pwResponse, err := m.grpcMarshaller.PointWrite(pointUUID, body)
	if err != nil {
		return nil, err
	}
	return &pwResponse.Point, nil
}

func (m *Module) internalPointUpdate(point *model.Point) (*model.Point, error) {
	pointWriter := &dto.PointWriter{
		OriginalValue: point.WriteValue,
		Message:       "",
		Fault:         false,
		PollState:     datatype.PointStatePollOk,
	}
	pwResponse, err := m.grpcMarshaller.PointWrite(point.UUID, pointWriter)
	if err != nil {
		log.Errorf("internalPointUpdate() error: %s", err)
		return nil, err
	}
	return &pwResponse.Point, nil
}

func (m *Module) handleSerialPayload(data string) {
	if m.networkUUID == "" {
		return
	}

	if !endec.ValidPayload(data) {
		return
	}

	var err error
	log.Debugf("uplink: %s", data)
	legacyDevice := false
	device := m.getDeviceByLoRaAddress(endec.DecodeAddress(data))

	if device == nil && !m.config.DecryptionDisabled {
		// maybe it's a legacy device (droplet, microedge)
		dataLegacy, err := decryptLegacy(data, m.config.DefaultKey)
		if err == nil {
			device = m.getDeviceByLoRaAddress(endec.DecodeAddress(dataLegacy))
			legacyDevice = true
			data = dataLegacy
		}
	}
	if device == nil {
		id := endec.DecodeAddress(data) // show user messages from lora
		rssi := endec.DecodeRSSI(data)
		log.Infof("message from non-added sensor. ID: %s, RSSI: %d", id, rssi)
		return
	}
	// Decode RSSI and SNR before decryption; otherwise, they will be lost.
	rssi := endec.DecodeRSSI(data)
	snr := endec.DecodeSNR(data)

	if !legacyDevice && !m.config.DecryptionDisabled {
		hexKey := m.config.DefaultKey
		if device.Manufacture != "" {
			hexKey = device.Manufacture // Manufacture property from device model holds hex key
		}
		data, err = decryptNormal(data, hexKey)
		if err != nil {
			log.Errorf("error decrypting data: %s", err)
			return
		}
	}

	err = decodeData(
		data,
		device,
		m.updateDevicePoint,
		m.updateDeviceMetaTags,
		m.pointWriteQueue.DequeueUsingMessageId,
		m.internalPointUpdate,
	)
	if err != nil {
		log.Errorf("decode error: %v\r\n", err)
		return
	}

	_ = m.updateDevicePoint(endec.RssiField, float64(rssi), device)
	_ = m.updateDevicePoint(endec.SnrField, float64(snr), device)

	m.updateDeviceFault(device.Model, device.UUID)
}

func decodeData(
	data string,
	device *model.Device,
	updatePointFn endec.UpdateDevicePointFunc,
	updateDeviceMetaTagFn endec.UpdateDeviceMetaTagsFunc,
	dequeuePointWriteFn endec.DequeuePointWriteFunc,
	internalPointUpdateFn endec.InternalPointUpdate,
) error {
	devDesc := endec.GetDeviceDescription(device)
	if devDesc == &endec.NilLoRaDeviceDescription {
		log.Errorln("nil device description found")
		return errors.New("no device description found")
	}

	if devDesc.IsLoRaRAW {
		/*
		 * Data Structure:
		 * ------------------------------------------------------------------------------------------------------------------------------------------------------------------------
		 * | 4 bytes address | 1 byte opts  | 1 byte nonce  | 1 byte length | Payload           | 4 bytes CMAC              | 1 bytes RSSI              |   1 bytes SNR           |
		 * ------------------------------------------------------------------------------------------------------------------------------------------------------------------------
		 * | data[0:3]       | data[4]      | data[5]       | data[6]       | data[7:dataLen-6] | data[dataLen-6:dataLen-2] | data[dataLen-2:dataLen-1] | data[dataLen-1:dataLen] |
		 * ------------------------------------------------------------------------------------------------------------------------------------------------------------------------
		 */

		if !utils.CheckLoRaRAWPayloadLength(data) {
			return errors.New("LoRaRaw payload length mismatched")
		}
		data = utils.StripLoRaRAWPayload(data)
	}

	if !devDesc.CheckLength(data) {
		return errors.New("invalid payload length")
	}

	err := endec.DecodePayload(
		data,
		devDesc,
		device,
		updatePointFn,
		updateDeviceMetaTagFn,
		dequeuePointWriteFn,
		internalPointUpdateFn,
	)
	return err
}

func decryptData(data string, hexKey string, decryptFunc func([]byte, []byte) ([]byte, error)) (string, error) {
	byteKey, err := hex.DecodeString(hexKey)
	if err != nil {
		log.Errorf("error decoding device key: %s", err)
		return "", err
	}
	byteData, err := hex.DecodeString(data)
	if err != nil {
		return "", err
	}
	decryptedData, err := decryptFunc(byteData[:len(byteData)-2], byteKey)
	if err != nil {
		return "", err
	}
	return strings.ToUpper(hex.EncodeToString(decryptedData)), nil
}

func decryptNormal(data string, hexKey string) (string, error) {
	return decryptData(data, hexKey, aesutils.Decrypt)
}

func decryptLegacy(data string, hexKey string) (string, error) {
	return decryptData(data, hexKey, aesutils.DecryptLegacy)
}

func (m *Module) getDeviceByLoRaAddress(address string) *model.Device {
	opts := &nmodule.Opts{Args: &nargs.Args{AddressUUID: &address, WithPoints: true, WithMetaTags: true}}
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

	points := endec.GetDevicePointNames(deviceBody)
	// TODO: should check this before the device is even added in the wizard
	if len(points) == 0 {
		log.Errorf("addDevicePoints() incorrect device model, try THLM %s", err)
		return errors.New("addDevicePoints() no device description or points found for this device")
	}
	m.addPointsFromName(deviceBody, points...)
	return nil
}

func (m *Module) addPointsFromName(deviceBody *model.Device, names ...string) {
	var points []*model.Point
	for _, name := range names {
		point := new(model.Point)
		setNewPointFields(deviceBody, point, name)
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
			if _, ok := field.Interface().(endec.CommonValues); !ok {
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
		setNewPointFields(deviceBody, point, pointName)
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

func (m *Module) getEncryptionKey(deviceUUID string) ([]byte, error) {
	device, err := m.grpcMarshaller.GetDevice(deviceUUID)
	if err != nil {
		return nil, err
	}

	hexKey := m.config.DefaultKey
	if device.Manufacture != "" {
		hexKey = device.Manufacture // Manufacture property from device model holds hex key
	}

	key, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, err
	}

	return key, nil
}
