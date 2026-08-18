package pkg

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/NubeIO/module-core-loraraw/schema"
	"github.com/NubeIO/module-core-loraraw/utils"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
)

// txRecorder captures frames handed to writeToLoRaRaw and, per device address,
// optionally acks them by feeding the messageId back into the manager the way
// the serial RX path does.
type txRecorder struct {
	mu     sync.Mutex
	frames []txFrame
	ackFor map[string]bool // address hex -> should ack
	mgr    *PointWriteQueueManager
	uuidBy map[string]string // address hex -> device uuid
	failN  int               // first N writes fail with a serial error
	notify chan txFrame
}

type txFrame struct {
	address string
	msgId   uint8
	at      time.Time
}

func (r *txRecorder) write(data []byte) error {
	r.mu.Lock()
	if r.failN > 0 {
		r.failN--
		r.mu.Unlock()
		return errors.New("serial down")
	}
	// The nonce/messageId sits inside the encrypted body, so read it off the
	// queue head (SetMessage runs before writeToLoRaRaw) instead of the wire.
	address := string(data[:utils.LORARAW_HEADER_LEN])
	uuid := r.uuidBy[address]
	var msgId uint8
	if head := r.mgr.getOrCreateQueue(uuid).Peek(); head != nil {
		msgId = head.MessageId
	}
	f := txFrame{address: address, msgId: msgId, at: time.Now()}
	r.frames = append(r.frames, f)
	ack := r.ackFor[f.address]
	r.mu.Unlock()

	if ack {
		go func() {
			time.Sleep(20 * time.Millisecond)
			r.mgr.DequeueUsingMessageId(uuid, f.msgId)
		}()
	}
	select {
	case r.notify <- f:
	default:
	}
	return nil
}

func (r *txRecorder) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.frames)
}

func (r *txRecorder) snapshot() []txFrame {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]txFrame(nil), r.frames...)
}

type schedFixture struct {
	rec       *txRecorder
	mgr       *PointWriteQueueManager
	devices   map[string]*model.Device
	exhausted []*model.Point
	exMu      sync.Mutex
}

func newSchedFixture(t *testing.T, maxRetry int, responseTimeout time.Duration, addresses ...string) *schedFixture {
	t.Helper()
	f := &schedFixture{devices: map[string]*model.Device{}}
	f.rec = &txRecorder{ackFor: map[string]bool{}, uuidBy: map[string]string{}, notify: make(chan txFrame, 64)}
	for _, addr := range addresses {
		a := addr
		uuid := "dev-" + a
		f.devices[uuid] = &model.Device{
			CommonUUID: model.CommonUUID{UUID: uuid},
			CommonDevice: model.CommonDevice{
				Model:       schema.DeviceModelUART,
				AddressUUID: &a,
			},
		}
		f.rec.uuidBy[addressBytes(a)] = uuid
	}
	getDevice := func(uuid string) (*model.Device, error) {
		d, ok := f.devices[uuid]
		if !ok {
			return nil, errors.New("device not found")
		}
		return d, nil
	}
	getKey := func(*model.Device) ([]byte, error) {
		return []byte("0123456789abcdef"), nil
	}
	f.mgr = NewPointWriteQueueManager(maxRetry, responseTimeout, getDevice, getKey, f.rec.write, func(p *model.Point) {
		f.exMu.Lock()
		f.exhausted = append(f.exhausted, p)
		f.exMu.Unlock()
	})
	f.rec.mgr = f.mgr
	t.Cleanup(f.mgr.Stop)
	return f
}

func addressBytes(hexAddr string) string {
	b := make([]byte, utils.LORARAW_HEADER_LEN)
	for i := 0; i < utils.LORARAW_HEADER_LEN; i++ {
		var v byte
		for _, c := range hexAddr[i*2 : i*2+2] {
			v <<= 4
			switch {
			case c >= '0' && c <= '9':
				v |= byte(c - '0')
			case c >= 'A' && c <= 'F':
				v |= byte(c-'A') + 10
			case c >= 'a' && c <= 'f':
				v |= byte(c-'a') + 10
			}
		}
		b[i] = v
	}
	return string(b)
}

func (f *schedFixture) point(addr, uuid string, value float64) *model.Point {
	a := addr
	return &model.Point{
		CommonUUID:  model.CommonUUID{UUID: uuid},
		IoNumber:    "UVP-43",
		DataType:    "30",
		DeviceUUID:  "dev-" + a,
		AddressUUID: &a,
		WriteValue:  &value,
	}
}

func (f *schedFixture) setAck(addr string, ack bool) {
	f.rec.mu.Lock()
	f.rec.ackFor[addressBytes(addr)] = ack
	f.rec.mu.Unlock()
}

func waitFor(t *testing.T, timeout time.Duration, cond func() bool) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return cond()
}

func TestScheduler_AckedWriteDequeuesImmediately(t *testing.T) {
	f := newSchedFixture(t, 5, 2*time.Second, "AAAAAAA1")
	f.setAck("AAAAAAA1", true)

	f.mgr.EnqueuePoint(f.point("AAAAAAA1", "p1", 2))

	if !waitFor(t, time.Second, func() bool { return f.rec.count() == 1 }) {
		t.Fatalf("expected 1 transmission, got %d", f.rec.count())
	}
	if !waitFor(t, time.Second, func() bool { return f.mgr.getOrCreateQueue("dev-AAAAAAA1").Size() == 0 }) {
		t.Fatalf("acked write should leave the queue")
	}
	time.Sleep(100 * time.Millisecond)
	if f.rec.count() != 1 {
		t.Fatalf("acked write must not be retransmitted, got %d transmissions", f.rec.count())
	}
}

func TestScheduler_HoldsRadioUntilResponseOrTimeout(t *testing.T) {
	const timeout = 300 * time.Millisecond
	f := newSchedFixture(t, 5, timeout, "AAAAAAA1", "BBBBBBB2")
	f.setAck("AAAAAAA1", false) // A never answers
	f.setAck("BBBBBBB2", true)

	f.mgr.EnqueuePoint(f.point("AAAAAAA1", "pA", 2))
	f.mgr.EnqueuePoint(f.point("BBBBBBB2", "pB", 3))

	if !waitFor(t, 2*time.Second, func() bool { return f.rec.count() >= 2 }) {
		t.Fatalf("expected both devices to transmit, got %d", f.rec.count())
	}
	frames := f.rec.snapshot()
	if frames[0].address != addressBytes("AAAAAAA1") || frames[1].address != addressBytes("BBBBBBB2") {
		t.Fatalf("expected A then B, got %x then %x", frames[0].address, frames[1].address)
	}
	gap := frames[1].at.Sub(frames[0].at)
	if gap < timeout {
		t.Fatalf("B transmitted %s after A; must wait the %s response timeout for A first", gap, timeout)
	}
	if gap > timeout+250*time.Millisecond {
		t.Fatalf("B transmitted %s after A; should go right after A's timeout", gap)
	}
}

func TestScheduler_RoundRobinAndExhaustion(t *testing.T) {
	const timeout = 100 * time.Millisecond
	f := newSchedFixture(t, 3, timeout, "AAAAAAA1", "BBBBBBB2")
	f.setAck("AAAAAAA1", false)
	f.setAck("BBBBBBB2", false)

	f.mgr.EnqueuePoint(f.point("AAAAAAA1", "pA", 2))
	f.mgr.EnqueuePoint(f.point("BBBBBBB2", "pB", 3))

	if !waitFor(t, 3*time.Second, func() bool { return f.rec.count() >= 6 }) {
		t.Fatalf("expected 3 attempts per device (6 total), got %d", f.rec.count())
	}
	frames := f.rec.snapshot()[:6]
	a, b := addressBytes("AAAAAAA1"), addressBytes("BBBBBBB2")
	want := []string{a, b, a, b, a, b}
	for i, fr := range frames {
		if fr.address != want[i] {
			t.Fatalf("attempt %d: expected %x got %x (not round-robin)", i, want[i], fr.address)
		}
	}
	// same messageId reused across retries of one point
	if frames[0].msgId != frames[2].msgId || frames[2].msgId != frames[4].msgId {
		t.Fatalf("retries must reuse the messageId: %v %v %v", frames[0].msgId, frames[2].msgId, frames[4].msgId)
	}

	if !waitFor(t, 2*time.Second, func() bool {
		f.exMu.Lock()
		defer f.exMu.Unlock()
		return len(f.exhausted) == 2
	}) {
		t.Fatalf("both points should be reported exhausted")
	}
	time.Sleep(200 * time.Millisecond)
	if f.rec.count() != 6 {
		t.Fatalf("no transmissions after exhaustion expected, got %d", f.rec.count())
	}
}

func TestScheduler_SameDeviceStaysOrdered(t *testing.T) {
	f := newSchedFixture(t, 5, time.Second, "AAAAAAA1")
	f.setAck("AAAAAAA1", true)

	f.mgr.EnqueuePoint(f.point("AAAAAAA1", "first", 1))
	f.mgr.EnqueuePoint(f.point("AAAAAAA1", "second", 2))

	if !waitFor(t, 2*time.Second, func() bool { return f.rec.count() == 2 }) {
		t.Fatalf("expected 2 transmissions, got %d", f.rec.count())
	}
	if !waitFor(t, time.Second, func() bool { return f.mgr.getOrCreateQueue("dev-AAAAAAA1").Size() == 0 }) {
		t.Fatalf("both writes should be acked and gone")
	}
}

func TestScheduler_SerialWriteErrorCountsAsAttempt(t *testing.T) {
	f := newSchedFixture(t, 2, 100*time.Millisecond, "AAAAAAA1")
	f.rec.mu.Lock()
	f.rec.failN = 100 // serial never comes back
	f.rec.mu.Unlock()

	f.mgr.EnqueuePoint(f.point("AAAAAAA1", "pA", 2))

	if !waitFor(t, 6*time.Second, func() bool {
		f.exMu.Lock()
		defer f.exMu.Unlock()
		return len(f.exhausted) == 1
	}) {
		t.Fatalf("write must be given up after maxRetry serial failures, not loop forever")
	}
}

func TestScheduler_UnknownDeviceIsDropped(t *testing.T) {
	f := newSchedFixture(t, 5, time.Second, "AAAAAAA1")
	p := f.point("AAAAAAA1", "ghost", 1)
	p.DeviceUUID = "no-such-device"

	f.mgr.EnqueuePoint(p)

	if !waitFor(t, time.Second, func() bool { return f.mgr.getOrCreateQueue("no-such-device").Size() == 0 }) {
		t.Fatalf("write for unknown device should be dropped")
	}
	if f.rec.count() != 0 {
		t.Fatalf("nothing should be transmitted for an unknown device")
	}
	f.exMu.Lock()
	defer f.exMu.Unlock()
	if len(f.exhausted) != 0 {
		t.Fatalf("dropped (not exhausted) write must not be reported as exhausted")
	}
}
