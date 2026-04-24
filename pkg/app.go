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
	"github.com/NubeIO/module-core-loraraw/schema"
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
	if body.AddressUUID != nil {
		*body.AddressUUID = strings.ToUpper(*body.AddressUUID)
	}
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
	log.Infof("handleSerialPayload: enter, networkUUID=%s, dataHex=%s", m.networkUUID, dataHex)

	if m.networkUUID == "" {
		log.Infof("handleSerialPayload: exit, no networkUUID set")
		return
	}

	if !codec.ValidPayload(dataHex) {
		log.Infof("handleSerialPayload: exit, invalid payload (length=%d)", len(dataHex))
		return
	}

	// We intentionally DO NOT publish the raw uplink here. When encryption is
	// enabled the on-the-wire bytes are encrypted and useless to other
	// consumers (only this module holds the key). Instead we publish the
	// decrypted/unwrapped frame further down, once we have it. For
	// non-encrypted paths the original hex is published as-is.
	publishRawHex := strings.ToUpper(dataHex)

	var err error
	log.Debugf("uplink: %s", dataHex)
	legacyDevice := false
	address, err := codec.DecodeAddressHex(dataHex)
	if err != nil {
		log.Errorf("failed to decode LoRa address from hex data (length=%d): %s", len(dataHex), err)
		return
	}
	log.Infof("handleSerialPayload: decoded address=%s", address)

	device := m.getDeviceByLoRaAddress(address)
	log.Infof("handleSerialPayload: initial device lookup: found=%v, enableDecryption=%v",
		device != nil, m.config.EnableDecryption)

	if device == nil && m.config.EnableDecryption {
		log.Infof("handleSerialPayload: attempting legacy decryption fallback for unknown address=%s", address)
		// maybe it's a legacy device (droplet, microedge, ziphydrotap)
		keyBytes, err := hex.DecodeString(m.config.DefaultKey)
		if err != nil {
			log.Errorf("error decoding default key: %s", err)
			return
		}
		dataBytesOrig, _ := hex.DecodeString(dataHex)
		dataLegacy, err := decryptLegacy(dataBytesOrig, keyBytes)
		if err == nil {
			log.Infof("handleSerialPayload: legacy decrypt succeeded, decryptedLen=%d", len(dataLegacy))
			address, err = codec.DecodeAddressBytes(dataLegacy)
			if err != nil {
				log.Errorf("failed to decode LoRa address from legacy bytes (length=%d): %s", len(dataLegacy), err)
			}
			address = strings.ToUpper(address)
			device = m.getDeviceByLoRaAddress(address)
			legacyDevice = true
			dataHex = hex.EncodeToString(dataLegacy)
			// Publish a raw frame that looks the way the wire looked before
			// the source started encrypting: decrypted body + original
			// rssi/snr trailer.
			if len(dataBytesOrig) >= 2 {
				pubBytes := append([]byte{}, dataLegacy...)
				pubBytes = append(pubBytes, dataBytesOrig[len(dataBytesOrig)-2:]...)
				publishRawHex = strings.ToUpper(hex.EncodeToString(pubBytes))
			} else {
				publishRawHex = strings.ToUpper(dataHex)
			}
			log.Infof("handleSerialPayload: legacy fallback resolved address=%s, deviceFound=%v", address, device != nil)
		} else {
			log.Infof("handleSerialPayload: legacy decrypt failed: %s", err)
		}
	}

	// Decode RSSI/SNR from the original frame NOW, before dataHex is potentially
	// replaced with the decrypted legacy payload (which has RSSI/SNR stripped off).
	rssi, err := codec.DecodeRSSI(dataHex)
	if err != nil {
		log.Errorf("failed to decode RSSI from hex data (address=%s, length=%d): %s", address, len(dataHex), err)
		return
	}
	snr, err := codec.DecodeSNR(dataHex)
	if err != nil {
		log.Errorf("failed to decode SNR from hex data (address=%s, length=%d): %s", address, len(dataHex), err)
		return
	}
	log.Infof("handleSerialPayload: address=%s rssi=%d snr=%.2f legacyDevice=%v", address, rssi, snr, legacyDevice)

	if device == nil {
		log.Infof("message from unknown sensor. ID: %s, RSSI: %d, SNR: %.2f", address, rssi, snr)
		return
	}
	devDesc := codec.GetDeviceDescription(device, codecs.LoRaDeviceDescriptions)
	if devDesc == &codec.NilLoRaDeviceDescription {
		log.Errorln("nil device description found")
		return
	}
	log.Infof("handleSerialPayload: matched device model=%s uuid=%s isLoRaRAW=%v",
		device.Model, device.UUID, devDesc.IsLoRaRAW)

	// Collect every decoded point value so we can publish them as a single
	// JSON payload over MQTT once decoding is complete.
	collected := map[string]float64{}
	successFn := func(name string, value float64, dev *model.Device, dd *codec.LoRaDeviceDescription) error {
		collected[name] = value
		return m.updateDevicePointSuccess(name, value, dev, dd)
	}

	if !legacyDevice && m.config.EnableDecryption {
		log.Infof("handleSerialPayload: taking LoRaRAW decrypt path for address=%s", address)
		keyBytes, err := m.getEncryptionKey(device)
		if err != nil {
			log.Errorf("error decoding default key: %s", err)
			return
		}
		dataBytes, _ := hex.DecodeString(dataHex)
		decodedDataBytes, err := decryptLoRaRAWPkt(dataBytes, keyBytes)
		if err != nil {
			log.Errorf("error decrypting data: (address: %s) %s", address, err)
			return
		}
		log.Infof("handleSerialPayload: LoRaRAW decrypt ok, decodedLen=%d, dispatching to LoRaRAW handler", len(decodedDataBytes))
		// Rebuild the frame as it would have appeared unencrypted on the wire:
		// [addr:4][opts:1][nonce:1][len:1][inner payload:len][cmac:4][rssi:1][snr:1]
		// This gives downstream MQTT consumers a stable, decodable frame that
		// does not depend on holding the encryption key.
		if pub, ok := buildUnencryptedRawFrame(decodedDataBytes, dataBytes); ok {
			publishRawHex = strings.ToUpper(hex.EncodeToString(pub))
		}
		m.handleLoRaRAWDevice(device, devDesc, dataHex, decodedDataBytes, keyBytes, successFn)
	} else if !legacyDevice && !m.config.EnableDecryption && devDesc.IsLoRaRAW {
		// Unencrypted LoRaRAW frame: same wire framing minus encryption and CMAC.
		// Layout: [addr:4][opts:1][nonce:1][len:1][payload:len][rssi:1][snr:1]
		log.Infof("handleSerialPayload: taking unencrypted LoRaRAW path for address=%s", address)
		dataBytes, _ := hex.DecodeString(dataHex)
		if len(dataBytes) < utils.LORARAW_PAYLOAD_START+2 {
			log.Errorf("unencrypted LoRaRAW frame too short: length %d, need at least %d",
				len(dataBytes), utils.LORARAW_PAYLOAD_START+2)
			return
		}
		innerLen := utils.GetLoRaRAWInnerPayloadLength(dataBytes)
		// Need: header(7) + payload(innerLen) + rssi/snr(2) bytes.
		if len(dataBytes) < utils.LORARAW_PAYLOAD_START+innerLen+2 {
			log.Errorf("unencrypted LoRaRAW frame truncated: need %d bytes, have %d",
				utils.LORARAW_PAYLOAD_START+innerLen+2, len(dataBytes))
			return
		}
		payload := utils.StripLoRaRAWPayload(dataBytes)
		log.Infof("handleSerialPayload: unencrypted LoRaRAW ok, innerLen=%d, dispatching to decoder", innerLen)
		if err := devDesc.DecodeUplink(dataHex, payload, devDesc, device,
			successFn, m.updateDevicePointError, m.updateDeviceMetaTags); err != nil {
			log.Errorf("error decoding unencrypted LoRaRAW uplink: %v", err)
		}
	} else {
		log.Infof("handleSerialPayload: taking legacy handler path (legacyDevice=%v, enableDecryption=%v)",
			legacyDevice, m.config.EnableDecryption)
		dataBytes, _ := hex.DecodeString(dataHex)
		m.handleLegacyDevice(device, devDesc, dataHex, dataBytes, successFn)
	}

	_ = successFn(codec.RssiField, float64(rssi), device, devDesc)
	_ = successFn(codec.SnrField, float64(snr), device, devDesc)
	m.updateDeviceFault(device.Model, device.UUID)
	log.Infof("handleSerialPayload: done for address=%s model=%s", address, device.Model)

	if m.mqttClient != nil {
		m.mqttClient.PublishRaw(publishRawHex)
		if device.AddressUUID != nil && len(collected) > 0 {
			m.mqttClient.PublishValues(*device.AddressUUID, device.Name, collected)
		}
	}
}

// buildUnencryptedRawFrame takes a decrypted LoRaRAW frame (as produced by
// decryptLoRaRAWPkt — i.e. addr+opts+nonce+len+padded_payload+cmac, WITHOUT
// rssi/snr) and the original on-the-wire bytes (which DO have rssi/snr as the
// last 2 bytes), and returns the frame rebuilt to match the pre-encryption
// wire layout that existed before the source enabled encryption:
//
//	[addr:4][opts:1][nonce:1][len:1][payload:len][rssi:1][snr:1]
//
// Notes:
//   - The AES padding after `payload` is stripped using the `len` field so the
//     published hex length is independent of the AES block size.
//   - The 4-byte CMAC is intentionally NOT included: it only exists to
//     authenticate the encrypted transport and is meaningless to downstream
//     MQTT consumers, which expect the legacy unencrypted layout.
//
// Returns false when the inputs are too short to reconstruct the frame safely.
func buildUnencryptedRawFrame(decoded, original []byte) ([]byte, bool) {
	if len(original) < 2 {
		return nil, false
	}
	if len(decoded) < utils.LORARAW_PAYLOAD_START {
		return nil, false
	}
	innerLen := utils.GetLoRaRAWInnerPayloadLength(decoded)
	if len(decoded) < utils.LORARAW_PAYLOAD_START+innerLen {
		return nil, false
	}
	out := make([]byte, 0, utils.LORARAW_PAYLOAD_START+innerLen+2)
	out = append(out, decoded[:utils.LORARAW_PAYLOAD_START+innerLen]...)
	// RSSI/SNR from the original wire frame (CMAC is intentionally dropped).
	out = append(out, original[len(original)-2:]...)
	return out, true
}

func (m *Module) handleLegacyDevice(device *model.Device, devDesc *codec.LoRaDeviceDescription, dataHex string, dataBytes []byte, successFn codec.UpdateDevicePointFunc) {
	if !devDesc.CheckLength(dataHex) {
		log.Errorf("invalid legacy payload length")
		return
	}

	err := devDesc.DecodeUplink(dataHex, dataBytes, devDesc, device, successFn, m.updateDevicePointError, m.updateDeviceMetaTags)
	if err != nil {
		log.Errorf("error decoding legacy uplink: %v", err)
	}
}

func (m *Module) handleLoRaRAWDevice(device *model.Device, devDesc *codec.LoRaDeviceDescription, dataHex string, dataBytes []byte, keyBytes []byte, successFn codec.UpdateDevicePointFunc) {
	if !utils.CheckLoRaRAWPayloadLength(dataBytes) {
		log.Errorf("LoRaRaw payload length mismatched")
		return
	}
	payload := utils.StripLoRaRAWPayload(dataBytes)

	opts := getOpts(dataBytes)
	switch opts {
	case utils.LORARAW_OPTS_CONFIRMED_UPLINK:
		m.handleConfirmedOpt(dataBytes, keyBytes)
		devDesc.DecodeUplink(dataHex, payload, devDesc, device, successFn, m.updateDevicePointError, m.updateDeviceMetaTags)
	case utils.LORARAW_OPTS_RESPONSE:
		if len(dataBytes) <= utils.LORARAW_NONCE_POSITION {
			log.Errorf("dataBytes too short for response: length %d, need at least %d", len(dataBytes), utils.LORARAW_NONCE_POSITION+1)
			return
		}
		msgId := dataBytes[utils.LORARAW_NONCE_POSITION]
		devDesc.DecodeResponse(dataHex, payload, msgId, devDesc, device, m.updateDeviceWrittenPointSuccess, m.updateDeviceWrittenPointError, m.updateDeviceMetaTags)
	default:
		log.Warnf("unhandled LoRaRAW option: %d", opts)
	}
}

func getOpts(dataBytes []byte) utils.LoRaRAWOpts {
	if len(dataBytes) <= utils.LORARAW_OPTS_POSITION {
		log.Errorf("dataBytes too short to get opts: length %d, need at least %d", len(dataBytes), utils.LORARAW_OPTS_POSITION+1)
		return utils.LoRaRAWOpts(0)
	}
	return utils.LoRaRAWOpts(dataBytes[utils.LORARAW_OPTS_POSITION])
}

func getNonce(dataBytes []byte) int {
	if len(dataBytes) <= utils.LORARAW_NONCE_POSITION {
		log.Errorf("dataBytes too short to get nonce: length %d, need at least %d", len(dataBytes), utils.LORARAW_NONCE_POSITION+1)
		return 0
	}
	return int(dataBytes[utils.LORARAW_NONCE_POSITION])
}

func (m *Module) handleConfirmedOpt(dataBytes []byte, byteKey []byte) {
	if len(dataBytes) < utils.LORARAW_HEADER_LEN {
		log.Errorf("dataBytes too short for confirmed opt: length %d, need at least %d", len(dataBytes), utils.LORARAW_HEADER_LEN)
		return
	}
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
	if len(dataBytes) < 2 {
		return nil, errors.New("dataBytes too short for decryption: need at least 2 bytes for RSSI and SNR")
	}
	decryptedData, err := aesutils.Decrypt(dataBytes[:len(dataBytes)-2], byteKey) // remove RSSI and SNR
	if err != nil {
		return nil, err
	}
	return decryptedData, nil
}

func decryptLegacy(dataBytes []byte, byteKey []byte) ([]byte, error) {
	if len(dataBytes) < 2 {
		return nil, errors.New("dataBytes too short for legacy decryption: need at least 2 bytes for RSSI and SNR")
	}
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
		// For UART devices, ensure RSSI and SNR have history enabled by default.
		if deviceBody.Model == schema.DeviceModelUART && (name == codec.RssiField || name == codec.SnrField) {
			setUARTCommonHistory(point)
		}
		point.EnableWriteable = boolean.NewFalse()
		points = append(points, point)
	}
	m.savePoints(points)
}

// setUARTCommonHistory configures default history for common values (e.g. RSSI/SNR) on UART devices.
func setUARTCommonHistory(pointBody *model.Point) {
	pointBody.HistoryEnable = boolean.NewTrue()
	pointBody.HistoryType = datatype.HistoryTypeInterval
	pointBody.HistoryInterval = integer.New(15)
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
