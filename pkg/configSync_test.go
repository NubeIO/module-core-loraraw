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

// The inclusive boundaries of the valid range (1..15000) must be accepted,
// not just rejected just outside them.
func TestResolveDesiredRateAcceptsBoundaries(t *testing.T) {
	for _, v := range []float64{1, 15000} {
		dev := &model.Device{Points: []*model.Point{
			pushRatePoint(nil, ratePtr(v)),
		}}
		got, ok := resolveDesiredRate(dev, pushRateIoNumber)
		if !ok {
			t.Fatalf("rate %v should have been accepted as within range", v)
		}
		if got != v {
			t.Fatalf("rate = %v, want %v", got, v)
		}
	}
}
