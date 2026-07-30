package pkg

import (
	"testing"

	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
)

func ratePtr(v float64) *float64 { return &v }

func pushRatePoint(present, write *float64) *model.Point {
	p := &model.Point{IoNumber: "UVP-1"}
	p.PresentValue = present
	p.WriteValue = write
	return p
}

// The operator's setpoint wins when present.
func TestResolveDesiredRatePrefersWriteValue(t *testing.T) {
	dev := &model.Device{Points: []*model.Point{
		pushRatePoint(ratePtr(600), ratePtr(900)),
	}}
	got, ok := resolveDesiredRate(dev, pushRateIoNumber)
	if !ok {
		t.Fatal("expected a rate to resolve")
	}
	if got != 900 {
		t.Fatalf("rate = %v, want 900 (the write value)", got)
	}
}

// With no setpoint the device is told to keep what it reported, so it gets a
// definitive answer instead of waiting out its RX window.
func TestResolveDesiredRateFallsBackToPresentValue(t *testing.T) {
	dev := &model.Device{Points: []*model.Point{
		pushRatePoint(ratePtr(600), nil),
	}}
	got, ok := resolveDesiredRate(dev, pushRateIoNumber)
	if !ok || got != 600 {
		t.Fatalf("rate = %v ok = %v, want 600 true", got, ok)
	}
}

func TestResolveDesiredRateMissingPoint(t *testing.T) {
	dev := &model.Device{Points: []*model.Point{}}
	if _, ok := resolveDesiredRate(dev, pushRateIoNumber); ok {
		t.Fatal("expected no rate when the point does not exist")
	}
}

func TestResolveDesiredRateRejectsOutOfRange(t *testing.T) {
	for _, v := range []float64{0, 15001, -5} {
		dev := &model.Device{Points: []*model.Point{
			pushRatePoint(nil, ratePtr(v)),
		}}
		if _, ok := resolveDesiredRate(dev, pushRateIoNumber); ok {
			t.Fatalf("rate %v should have been rejected as out of range", v)
		}
	}
}

// The response body must carry the response flag and echo the request's MID.
func TestBuildConfigResponsePayload(t *testing.T) {
	body, err := buildConfigResponsePayload(0x5A, 900)
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

// Response caching (§2.2) is deliberately not implemented, which is only safe
// if answering twice has no extra side effect. Clearing the pending write is
// the one mutation, so it must be a no-op the second time.
func TestDequeueByIoNumberIsIdempotent(t *testing.T) {
	mgr := NewPointWriteQueueManager(1, 0, nil, nil, nil, nil)
	mgr.EnqueuePoint(&model.Point{IoNumber: "UVP-1", DeviceUUID: "dev-1"})

	if got := mgr.DequeueByIoNumber("dev-1", "UVP-1"); got == nil {
		t.Fatal("first dequeue should return the queued point")
	}
	if got := mgr.DequeueByIoNumber("dev-1", "UVP-1"); got != nil {
		t.Fatalf("second dequeue should return nil, got %v", got)
	}
}

func TestDequeueByIoNumberUnknownDevice(t *testing.T) {
	mgr := NewPointWriteQueueManager(1, 0, nil, nil, nil, nil)
	if got := mgr.DequeueByIoNumber("nope", "UVP-1"); got != nil {
		t.Fatalf("expected nil for an unknown device, got %v", got)
	}
}

func TestDequeueByIoNumberLeavesOtherPoints(t *testing.T) {
	mgr := NewPointWriteQueueManager(1, 0, nil, nil, nil, nil)
	mgr.EnqueuePoint(&model.Point{IoNumber: "UVP-2", DeviceUUID: "dev-1"})
	mgr.EnqueuePoint(&model.Point{IoNumber: "UVP-1", DeviceUUID: "dev-1"})

	if got := mgr.DequeueByIoNumber("dev-1", "UVP-1"); got == nil {
		t.Fatal("expected UVP-1 to be dequeued")
	}
	if got := mgr.DequeueByIoNumber("dev-1", "UVP-2"); got == nil {
		t.Fatal("UVP-2 should still be queued")
	}
}
