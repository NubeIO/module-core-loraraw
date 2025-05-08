package pkg

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/NubeIO/lib-module-go/nmodule"
	"github.com/NubeIO/lib-utils-go/boolean"
	"github.com/NubeIO/lib-utils-go/integer"
	"github.com/NubeIO/module-core-loraraw/aesutils"
	"github.com/NubeIO/module-core-loraraw/codec"
	"github.com/NubeIO/module-core-loraraw/codecs"
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
	body.PollState = datatype.PointStateApiWritePending
	pwResponse, err := m.grpcMarshaller.PointWrite(pointUUID, body)
	if err != nil {
		return nil, err
	}
	return &pwResponse.Point, nil
}

func (m *Module) handleSerialPayload(dataHex string) {
	if m.networkUUID == "" {
		return
	}

	if !codec.ValidPayload(dataHex) {
		return
	}

	var err error
	log.Debugf("uplink: %s", dataHex)
	legacyDevice := false
	address := codec.DecodeAddressHex(dataHex)
	device := m.getDeviceByLoRaAddress(address)

	dataBytes, err := hex.DecodeString(dataHex)
	if err != nil {
		log.Errorf("error decoding data: (address: %s) %s", address, err)
		return
	}

	if device == nil && !m.config.DecryptionDisabled {
		// maybe it's a legacy device (droplet, microedge, ziphydrotap)
		keyBytes, err := hex.DecodeString(m.config.DefaultKey)
		if err != nil {
			log.Errorf("error decoding default key: %s", err)
			return
		}
		dataLegacy, err := decryptLegacy(dataBytes, keyBytes)
		if err == nil {
			address = codec.DecodeAddressBytes(dataLegacy)
			device = m.getDeviceByLoRaAddress(address)
			legacyDevice = true
			dataHex = hex.EncodeToString(dataLegacy)
			dataBytes = dataLegacy
		}
	}

	rssi := codec.DecodeRSSI(dataHex)
	snr := codec.DecodeSNR(dataHex)

	if device == nil {
		log.Infof("message from unknown sensor. ID: %s, RSSI: %d, SNR: %d", address, rssi, snr)
		return
	}
	devDesc := codec.GetDeviceDescription(device, codecs.LoRaDeviceDescriptions)
	if devDesc == &codec.NilLoRaDeviceDescription {
		log.Errorln("nil device description found")
		return
	}

	if !legacyDevice && !m.config.DecryptionDisabled {
		keyBytes, err := m.getEncryptionKey(device)
		if err != nil {
			log.Errorf("error decoding default key: %s", err)
			return
		}
		dataBytes, err = decryptLoRaRAWPkt(dataBytes, keyBytes)
		if err != nil {
			log.Errorf("error decrypting data: (address: %s) %s", address, err)
			return
		}
		m.handleLoRaRAWDevice(device, devDesc, dataHex, dataBytes, keyBytes)
	} else {
		m.handleLegacyDevice(device, devDesc, dataHex, dataBytes)
	}

	m.updateDevicePointSuccess(codec.RssiField, float64(rssi), device)
	m.updateDevicePointSuccess(codec.SnrField, float64(snr), device)
	m.updateDeviceFault(device.Model, device.UUID)
}

func (m *Module) handleLegacyDevice(device *model.Device, devDesc *codec.LoRaDeviceDescription, dataHex string, dataBytes []byte) {
	if !devDesc.CheckLength(dataHex) {
		log.Errorf("invalid legacy payload length")
		return
	}

	err := devDesc.DecodeUplink(dataHex, dataBytes, devDesc, device, m.updateDevicePointSuccess, m.updateDevicePointError, m.updateDeviceMetaTags)
	if err != nil {
		log.Errorf("error decoding legacy uplink: %v", err)
	}
}

func (m *Module) handleLoRaRAWDevice(device *model.Device, devDesc *codec.LoRaDeviceDescription, dataHex string, dataBytes []byte, keyBytes []byte) {
	if !utils.CheckLoRaRAWPayloadLength(dataBytes) {
		log.Errorf("LoRaRaw payload length mismatched")
		return
	}
	dataBytes = utils.StripLoRaRAWPayload(dataBytes)

	opts := getOpts(dataBytes)
	switch opts {
	case utils.LORARAW_OPTS_CONFIRMED_UPLINK:
		m.handleConfirmedOpt(dataBytes, keyBytes)
		devDesc.DecodeUplink(dataHex, dataBytes, devDesc, device, m.updateDevicePointSuccess, m.updateDevicePointError, m.updateDeviceMetaTags)
	case utils.LORARAW_OPTS_RESPONSE:
		devDesc.DecodeResponse(dataHex, dataBytes, devDesc, device, m.updateDeviceWrittenPointSuccess, m.updateDeviceWrittenPointError, m.updateDeviceMetaTags)
	default:
		log.Warnf("unhandled LoRaRAW option: %d", opts)
	}
}

func getOpts(dataBytes []byte) utils.LoRaRAWOpts {
	return utils.LoRaRAWOpts(dataBytes[utils.LORARAW_OPTS_POSITION])
}

func getNonce(dataBytes []byte) int {
	return int(dataBytes[utils.LORARAW_NONCE_POSITION])
}

func (m *Module) handleConfirmedOpt(dataBytes []byte, byteKey []byte) {
	ack := createAck(dataBytes[:utils.LORARAW_HEADER_LEN], byteKey, getNonce(dataBytes))
	err := m.WriteToLoRaRaw(ack)
	if err != nil {
		log.Errorf("error sending acknowledgement: %s", err)
	}
}

func createAck(address, key []byte, nonce int) []byte {
	optB := []byte{byte(utils.LORARAW_OPTS_ACK)}
	nonceB := []byte{byte(nonce)}
	var buffer bytes.Buffer
	buffer.Write(address)
	buffer.Write(optB)
	buffer.Write(nonceB)
	fullData := buffer.Bytes()
	unCmac, err := aesutils.CmacUnencrypted(fullData, key)
	if err != nil {
		log.Errorf("error creating ack: %s", err)
		return nil
	}
	fullData = append(fullData, unCmac...)
	return fullData
}

func decryptLoRaRAWPkt(dataBytes []byte, byteKey []byte) ([]byte, error) {
	decryptedData, err := aesutils.Decrypt(dataBytes[:len(dataBytes)-2], byteKey) // remove RSSI and SNR
	if err != nil {
		return nil, err
	}
	return decryptedData, nil
}

func decryptLegacy(dataBytes []byte, byteKey []byte) ([]byte, error) {
	decryptedData, err := aesutils.DecryptLegacy(dataBytes[:len(dataBytes)-2], byteKey) // remove RSSI and SNR
	if err != nil {
		return nil, err
	}
	return decryptedData, nil
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

	points := codec.GetDevicePointNames(deviceBody, codecs.LoRaDeviceDescriptions)
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
			if _, ok := field.Interface().(codec.CommonValues); !ok {
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

func (m *Module) getDevice(uuid string) (*model.Device, error) {
	device, err := m.grpcMarshaller.GetDevice(uuid)
	if err != nil {
		return nil, err
	}
	return device, nil
}

func (m *Module) getEncryptionKey(device *model.Device) ([]byte, error) {
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

func (m *Module) initWriteQueue() {
	m.writeQueueInit.Do(func() {
		m.writeQueue = make(chan []byte, 100)
		m.writeQueueDone = make(chan struct{})

		go m.processWriteQueue()
	})
}

func (m *Module) processWriteQueue() {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("Recovered panic in processWriteQueue: %v", r)
			// Restart goroutine
			go m.processWriteQueue()
		}
	}()

	for {
		select {
		case data := <-m.writeQueue:
			if Port == nil {
				log.Error("Serial port not connected")
				continue
			}

			_, err := Port.Write(data)
			if err != nil {
				log.Errorf("Error writing to serial port: %v", err)
			}

			// Wait a while after sending for the LoRa module to process
			time.Sleep(50 * time.Millisecond)

		case <-m.writeQueueDone:
			return
		}
	}
}
