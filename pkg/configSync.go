package pkg

import (
	"strconv"

	"github.com/NubeIO/module-core-loraraw/codecs/rubixDataEncoding"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
	log "github.com/sirupsen/logrus"
)

// Configuration sync is one user of the LoRaRAW request/response exchange, not
// the exchange itself - see loraRawRequest.go for the mechanism. Everything in
// this file is specific to the push rate.

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

// resolvePushRatePoint packages the resolved rate as the point the generic
// request handler will encode. MDK_UINT_16 rather than MDK_PUSH_FREQUENCY:
// that key's 0..2000 range is narrower than the 1..15000 the firmware accepts.
func resolvePushRatePoint(device *model.Device) (*model.Point, bool) {
	rate, ok := resolveDesiredRate(device, pushRateIoNumber)
	if !ok {
		return nil, false
	}
	value := rate
	return &model.Point{
		IoNumber:   pushRateIoNumber,
		DataType:   strconv.Itoa(int(rubixDataEncoding.MDK_UINT_16)),
		WriteValue: &value,
	}, true
}
