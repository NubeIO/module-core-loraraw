package pkg

import (
	"errors"

	"github.com/NubeIO/lib-utils-go/nstring"
	"github.com/NubeIO/module-core-loraraw/aesutils"
	"github.com/NubeIO/module-core-loraraw/codec"
	"github.com/NubeIO/module-core-loraraw/codecs/rubixDataEncoding"
	"github.com/NubeIO/module-core-loraraw/utils"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
	log "github.com/sirupsen/logrus"
)

// The LoRaRAW request/response exchange (§2.2) is a general mechanism: a device
// names the points it wants and the gateway answers with their values. Nothing
// about it is specific to configuration - configuration just happens to be the
// first thing a device asks for. Anything that knows what an individual point
// means belongs behind resolveRequestedPoint, not in this file.

// resolveRequestedPoint returns the point to answer with for ioNumber, or false
// when this gateway has nothing to say about it. This is the single hook where
// per-point knowledge lives; new request-able points are added here.
func resolveRequestedPoint(device *model.Device, ioNumber string) (*model.Point, bool) {
	switch ioNumber {
	case pushRateIoNumber:
		return resolvePushRatePoint(device)
	default:
		return nil, false
	}
}

// buildResponsePayload encodes a §2.2 response body:
//
//	[SETTINGS_BYTE with response flag] [RDE message ID] [POINT_ID][DATA_TYPE_ID][value]...
func buildResponsePayload(msgID uint8, points []*model.Point) ([]byte, error) {
	if len(points) == 0 {
		return nil, errors.New("no points to encode into a response")
	}

	body, err := rubixDataEncoding.EncodeRequestMessage(points)
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

// handleInboundRequest answers a device's LORARAW_OPTS_REQUEST. It replies
// synchronously via WriteToLoRaRaw - never through the write queue, whose
// time-off-air sleep would miss the device's ~1s RX window.
func (m *Module) handleInboundRequest(
	device *model.Device,
	_ *codec.LoRaDeviceDescription,
	payload []byte,
	dataBytes []byte,
	keyBytes []byte,
) {
	if len(dataBytes) <= utils.LORARAW_NONCE_POSITION {
		log.Errorf("loraRawRequest: frame too short for a request: length %d, need at least %d",
			len(dataBytes), utils.LORARAW_NONCE_POSITION+1)
		return
	}
	msgID := dataBytes[utils.LORARAW_NONCE_POSITION]

	requested, err := rubixDataEncoding.DecodeRequestPayload(payload)
	if err != nil {
		log.Errorf("loraRawRequest: cannot decode request: %s", err)
		return
	}
	log.Infof("loraRawRequest: device %s requested %v (mid=%d)", device.UUID, requested, msgID)

	// Answer what was asked for rather than a fixed point: a device asking for
	// something this gateway does not serve gets that entry dropped, not a
	// value it never requested.
	points := make([]*model.Point, 0, len(requested))
	for _, ioNumber := range requested {
		point, ok := resolveRequestedPoint(device, ioNumber)
		if !ok {
			log.Warnf("loraRawRequest: nothing to answer for %s on device %s", ioNumber, device.UUID)
			continue
		}
		points = append(points, point)
	}
	if len(points) == 0 {
		log.Warnf("loraRawRequest: no answerable points for device %s, not responding", device.UUID)
		return
	}

	body, err := buildResponsePayload(msgID, points)
	if err != nil {
		log.Errorf("loraRawRequest: cannot encode response: %s", err)
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
		log.Errorf("loraRawRequest: cannot encrypt response: %s", err)
		return
	}
	if err := m.WriteToLoRaRaw(frame); err != nil {
		log.Errorf("loraRawRequest: cannot send response: %s", err)
		return
	}
	log.Infof("loraRawRequest: answered device %s with %d point(s) (mid=%d)", device.UUID, len(points), msgID)
}
