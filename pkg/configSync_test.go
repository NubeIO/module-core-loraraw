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

// Regression test for a review finding: ProcessPointWriteQueue used to remove
// its finished item with a blind pop-front (DequeueWriteQueue), which assumed
// the front of the slice was still the item it took. DequeueByIoNumber breaks
// that assumption because it can remove an item mid-slice while the worker is
// off doing external work (encode/encrypt/transmit/sleep) on a different
// item. This reproduces that interleaving directly against the queue
// internals and asserts the worker's removal is identity-based, so it never
// discards an unrelated, never-transmitted point.
func TestWorkerRemovalSurvivesConcurrentDequeueByIoNumber(t *testing.T) {
	mgr := NewPointWriteQueueManager(1, 0, nil, nil, nil, nil)

	// Insert the queue directly instead of going through EnqueuePoint: that
	// path spins up a background ProcessPointWriteQueue goroutine which,
	// finding the queue non-empty, would try to process the point left
	// behind at the end of this test using the nil getDevice/getEncryptionKey
	// funcs above and panic. Driving the queue by hand keeps this test
	// deterministic and focused on the removal-ordering invariant.
	queue := NewPointWriteQueue()
	mgr.mutex.Lock()
	mgr.queues["dev-1"] = queue
	mgr.mutex.Unlock()

	queue.EnqueueWriteQueue(&model.Point{IoNumber: "UVP-1", DeviceUUID: "dev-1"}) // A: push-rate write
	queue.EnqueueWriteQueue(&model.Point{IoNumber: "UVP-2", DeviceUUID: "dev-1"}) // B: unrelated write

	// Simulate ProcessPointWriteQueue taking the front item (A) and
	// releasing the lock to do external work, exactly as
	// `pendingPointWrite := pwq.writeQueue[0]; pwq.mutex.Unlock()` does.
	queue.mutex.Lock()
	pendingPointWrite := queue.writeQueue[0]
	queue.mutex.Unlock()
	if pendingPointWrite.Point.IoNumber != "UVP-1" {
		t.Fatalf("test setup broken: expected UVP-1 at the front, got %s", pendingPointWrite.Point.IoNumber)
	}

	// While the worker holds A, a config response arrives for the push rate
	// and settles it via DequeueByIoNumber.
	settled := mgr.DequeueByIoNumber("dev-1", "UVP-1")
	if settled == nil || settled.IoNumber != "UVP-1" {
		t.Fatalf("expected DequeueByIoNumber to settle UVP-1, got %v", settled)
	}

	// The worker now finishes its own item and removes exactly what it
	// processed, mirroring ProcessPointWriteQueue's removal sites.
	queue.removePendingWrite(pendingPointWrite)

	// B must still be queued.
	queue.mutex.Lock()
	defer queue.mutex.Unlock()
	if len(queue.writeQueue) != 1 {
		t.Fatalf("expected 1 point left in queue, got %d", len(queue.writeQueue))
	}
	if queue.writeQueue[0].Point.IoNumber != "UVP-2" {
		t.Fatalf("expected UVP-2 to survive, got %q", queue.writeQueue[0].Point.IoNumber)
	}
}
