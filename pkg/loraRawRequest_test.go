package pkg

import (
	"testing"

	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
)

// The response body must carry the response flag and echo the request's MID.
func TestBuildResponsePayload(t *testing.T) {
	point, ok := resolvePushRatePoint(&model.Device{Points: []*model.Point{
		pushRatePoint(nil, ratePtr(900)),
	}})
	if !ok {
		t.Fatal("expected the push rate point to resolve")
	}

	body, err := buildResponsePayload(0x5A, []*model.Point{point})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(body) < 3 {
		t.Fatalf("body too short: %d bytes", len(body))
	}
	if body[0]&4 != 4 {
		t.Fatalf("settings byte %#x does not have the response flag set", body[0])
	}
	if body[1] != 0x5A {
		t.Fatalf("message ID = %#x, want 0x5A", body[1])
	}
}

// Encoding nothing is an error rather than an empty frame: a response with no
// points would leave the device waiting out its RX window for a value that is
// not there.
func TestBuildResponsePayloadRejectsNoPoints(t *testing.T) {
	if _, err := buildResponsePayload(0x5A, nil); err == nil {
		t.Fatal("expected an error when there are no points to encode")
	}
}

// The mechanism is not config-specific, but it only answers points it has a
// resolver for - anything else is dropped rather than answered with a value
// the device never asked about.
func TestResolveRequestedPoint(t *testing.T) {
	dev := &model.Device{Points: []*model.Point{
		pushRatePoint(nil, ratePtr(900)),
	}}

	if _, ok := resolveRequestedPoint(dev, pushRateIoNumber); !ok {
		t.Fatalf("expected %s to resolve", pushRateIoNumber)
	}
	if _, ok := resolveRequestedPoint(dev, "UVP-40"); ok {
		t.Fatal("expected an unserved point to be refused, not answered")
	}
}
