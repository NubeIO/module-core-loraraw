package pkg

import (
	"errors"
	"strconv"

	"github.com/NubeIO/lib-utils-go/nstring"
	"github.com/NubeIO/module-core-loraraw/aesutils"
	"github.com/NubeIO/module-core-loraraw/codec"
	"github.com/NubeIO/module-core-loraraw/codecs/rubixDataEncoding"
	"github.com/NubeIO/module-core-loraraw/utils"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
	log "github.com/sirupsen/logrus"
)

// pushRateIoNumber is the point the device uses for its push rate (tdc_s).
// Future config items start at UVP-40 to avoid colliding with telemetry slots.
const pushRateIoNumber = "UVP-1"

// Bounds enforced by AT_TDC_set in the firmware (lora_at.c). Keep in step.
const (
	minPushRateSeconds = 1
	maxPushRateSeconds = 15000
)

// resolveDesiredRate returns the push rate to send back for ioNumber. The
// operator's WriteValue wins; otherwise the device is echoed its own last
// reported value so it gets a definitive answer rather than waiting out its
// RX window. Returns false when the point is absent or has no usable value.
func resolveDesiredRate(device *model.Device, ioNumber string) (float64, bool) {
	if device == nil {
		return 0, false
	}
	for _, pnt := range device.Points {
		if pnt == nil || pnt.IoNumber != ioNumber {
			continue
		}
		for _, candidate := range []*float64{pnt.WriteValue, pnt.PresentValue} {
			if candidate == nil {
				continue
			}
			v := *candidate
			if v < minPushRateSeconds || v > maxPushRateSeconds {
				log.Warnf("configSync: push rate %v for %s is outside %d..%d, ignoring",
					v, ioNumber, minPushRateSeconds, maxPushRateSeconds)
				continue
			}
			return v, true
		}
		return 0, false
	}
	return 0, false
}

// buildConfigResponsePayload encodes a §2.2 response body:
//
//	[SETTINGS_BYTE with response flag] [RDE message ID] [POINT_ID][DATA_TYPE_ID][value]
func buildConfigResponsePayload(msgID uint8, rate float64) ([]byte, error) {
	value := rate
	point := &model.Point{
		IoNumber:   pushRateIoNumber,
		DataType:   strconv.Itoa(int(rubixDataEncoding.MDK_UINT_16)),
		WriteValue: &value,
	}
	body, err := rubixDataEncoding.EncodeRequestMessage([]*model.Point{point})
	if err != nil {
		return nil, err
	}
	if len(body) < 1 {
		return nil, errors.New("encoder produced an empty response body")
	}

	// EncodeRequestMessage emits [settings][data...]. Splice in the response
	// flag and the RDE message ID, which must sit at index 1.
	sd := rubixDataEncoding.NewSerialDataWithBuffer([]byte{body[0]})
	rubixDataEncoding.SetResponseData(sd, true)
	rubixDataEncoding.SetMessageId(sd, msgID)

	out := make([]byte, 0, len(body)+1)
	out = append(out, sd.Buffer...) // [settings][mid]
	out = append(out, body[1:]...)  // data packets
	return out, nil
}

// handleConfigRequest answers a device's §2.2 config request. It replies
// synchronously via WriteToLoRaRaw — never through the write queue, whose
// time-off-air sleep would miss the device's ~1s RX window.
func (m *Module) handleConfigRequest(
	device *model.Device,
	_ *codec.LoRaDeviceDescription,
	payload []byte,
	dataBytes []byte,
	keyBytes []byte,
) {
	if len(dataBytes) <= utils.LORARAW_NONCE_POSITION {
		log.Errorf("configSync: frame too short for a request: length %d, need at least %d",
			len(dataBytes), utils.LORARAW_NONCE_POSITION+1)
		return
	}
	msgID := dataBytes[utils.LORARAW_NONCE_POSITION]

	requested, err := rubixDataEncoding.DecodeConfigRequest(payload)
	if err != nil {
		log.Errorf("configSync: cannot decode request: %s", err)
		return
	}
	log.Infof("configSync: device %s requested %v (mid=%d)", device.UUID, requested, msgID)

	rate, ok := resolveDesiredRate(device, pushRateIoNumber)
	if !ok {
		log.Warnf("configSync: no usable push rate for device %s, not responding", device.UUID)
		return
	}

	body, err := buildConfigResponsePayload(msgID, rate)
	if err != nil {
		log.Errorf("configSync: cannot encode response: %s", err)
		return
	}

	frame, err := aesutils.Encrypt(
		nstring.DerefString(device.AddressUUID),
		body,
		keyBytes,
		utils.LORARAW_OPTS_RESPONSE,
		msgID,
	)
	if err != nil {
		log.Errorf("configSync: cannot encrypt response: %s", err)
		return
	}
	if err := m.WriteToLoRaRaw(frame); err != nil {
		log.Errorf("configSync: cannot send response: %s", err)
		return
	}
	log.Infof("configSync: answered device %s with push rate %v (mid=%d)", device.UUID, rate, msgID)
}
