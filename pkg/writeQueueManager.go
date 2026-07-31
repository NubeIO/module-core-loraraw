package pkg

import (
	"errors"
	"sync"
	"time"

	"github.com/NubeIO/lib-utils-go/nstring"
	"github.com/NubeIO/module-core-loraraw/aesutils"
	"github.com/NubeIO/module-core-loraraw/codec"
	"github.com/NubeIO/module-core-loraraw/codecs"
	"github.com/NubeIO/module-core-loraraw/utils"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
	log "github.com/sirupsen/logrus"
)

// serialWriteErrorBackoff is how long the scheduler pauses after the serial
// port rejects a frame before it tries the next one.
const serialWriteErrorBackoff = 2 * time.Second

// ---------------------------------------------------
// MANAGER — one queue per DeviceUUID, one scheduler for the radio.
//
// The LoRa radio is half-duplex: while it transmits it cannot hear a device's
// RESPONSE. So exactly one frame is on air at a time, and after each
// transmission the scheduler waits for that frame's RESPONSE (or the response
// timeout) before it moves on to the next device's frame, round-robin.
// ---------------------------------------------------

type PointWriteQueueManager struct {
	queues          map[string]*PointWriteQueue
	order           []string
	rrIndex         int
	mutex           sync.Mutex
	maxRetry        int
	responseTimeout time.Duration

	notify chan struct{}
	stop   chan struct{}
	once   sync.Once

	getDevice        func(string) (*model.Device, error)
	getEncryptionKey func(*model.Device) ([]byte, error)
	writeToLoRaRaw   func([]byte) error
	onWriteExhausted func(*model.Point)
}

func NewPointWriteQueueManager(
	maxRetry int,
	responseTimeout time.Duration,
	getDevice func(string) (*model.Device, error),
	getEncryptionKey func(*model.Device) ([]byte, error),
	writeToLoRaRaw func([]byte) error,
	onWriteExhausted func(*model.Point),
) *PointWriteQueueManager {
	m := &PointWriteQueueManager{
		queues:           make(map[string]*PointWriteQueue),
		maxRetry:         maxRetry,
		responseTimeout:  responseTimeout,
		notify:           make(chan struct{}, 1),
		stop:             make(chan struct{}),
		getDevice:        getDevice,
		getEncryptionKey: getEncryptionKey,
		writeToLoRaRaw:   writeToLoRaRaw,
		onWriteExhausted: onWriteExhausted,
	}
	go m.schedule()
	return m
}

// Stop ends the scheduler goroutine. Safe to call more than once.
func (m *PointWriteQueueManager) Stop() {
	m.once.Do(func() { close(m.stop) })
}

func (m *PointWriteQueueManager) getOrCreateQueue(deviceUUID string) *PointWriteQueue {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	queue, exists := m.queues[deviceUUID]
	if !exists {
		queue = NewPointWriteQueue()
		m.queues[deviceUUID] = queue
		m.order = append(m.order, deviceUUID)
		log.Infof("created new write queue for device %s", deviceUUID)
	}
	return queue
}

func (m *PointWriteQueueManager) EnqueuePoint(point *model.Point) {
	queue := m.getOrCreateQueue(point.DeviceUUID)
	queue.EnqueueWriteQueue(point)
	m.wake()
}

func (m *PointWriteQueueManager) wake() {
	select {
	case m.notify <- struct{}{}:
	default:
	}
}

func (m *PointWriteQueueManager) DequeueUsingMessageId(deviceUUID string, messageId uint8) *model.Point {
	m.mutex.Lock()
	queue, exists := m.queues[deviceUUID]
	m.mutex.Unlock()
	if !exists {
		return nil
	}

	queue.mutex.Lock()
	defer queue.mutex.Unlock()

	item := queue.dequeue(&messageId)
	if item == nil {
		log.Warnf("[%s] no pending point write found for messageId %v", deviceUUID, messageId)
		return nil
	}
	return item.Point
}

// nextPending picks the next non-empty queue round-robin. Returns nil when
// nothing is pending anywhere.
func (m *PointWriteQueueManager) nextPending() (string, *PointWriteQueue, *PendingPointWrite) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	n := len(m.order)
	for i := 0; i < n; i++ {
		idx := (m.rrIndex + i) % n
		deviceUUID := m.order[idx]
		queue := m.queues[deviceUUID]
		if item := queue.Peek(); item != nil {
			m.rrIndex = idx + 1 // not modded: order may grow before the next pick
			return deviceUUID, queue, item
		}
	}
	return "", nil, nil
}

func (m *PointWriteQueueManager) schedule() {
	for {
		deviceUUID, queue, item := m.nextPending()
		if item == nil {
			select {
			case <-m.notify:
				continue
			case <-m.stop:
				return
			}
		}

		select {
		case <-m.stop:
			return
		default:
		}

		m.transmit(deviceUUID, queue, item)
	}
}

// transmit sends one attempt of the item and blocks until it is acked or the
// response timeout elapses.
func (m *PointWriteQueueManager) transmit(deviceUUID string, queue *PointWriteQueue, item *PendingPointWrite) {
	if item.Message == nil {
		if err := m.prepareMessage(queue, item); err != nil {
			// Device gone, bad key or unencodable point: nothing to retry.
			log.Errorf("[%s] dropping write for point %s: %s", deviceUUID, item.Point.UUID, err.Error())
			queue.RemoveItem(item)
			return
		}
	}

	if err := m.writeToLoRaRaw(item.Message); err != nil {
		log.Errorf("[%s] error writing to LoRa serial port: %v", deviceUUID, err)
		m.finishAttempt(deviceUUID, queue, item)
		time.Sleep(serialWriteErrorBackoff)
		return
	}

	timer := time.NewTimer(m.responseTimeout)
	defer timer.Stop()
	select {
	case <-item.done:
		if item.acked {
			log.Infof("[%s] write acked for point %s (messageId %d)", deviceUUID, item.Point.UUID, item.MessageId)
		}
		return
	case <-timer.C:
		log.Warnf("[%s] no response for point %s (messageId %d) within %s", deviceUUID, item.Point.UUID, item.MessageId, m.responseTimeout)
		m.finishAttempt(deviceUUID, queue, item)
	case <-m.stop:
		return
	}
}

// finishAttempt records a failed attempt and gives up on the item once
// maxRetry attempts have been made.
func (m *PointWriteQueueManager) finishAttempt(deviceUUID string, queue *PointWriteQueue, item *PendingPointWrite) {
	if queue.IncRetry(item) < m.maxRetry {
		return
	}
	if !queue.RemoveItem(item) {
		return
	}
	log.Warnf("[%s] write to point %s exhausted after %d attempts", deviceUUID, item.Point.UUID, m.maxRetry)
	if m.onWriteExhausted != nil {
		m.onWriteExhausted(item.Point)
	}
}

func (m *PointWriteQueueManager) prepareMessage(queue *PointWriteQueue, item *PendingPointWrite) error {
	device, err := m.getDevice(item.Point.DeviceUUID)
	if err != nil {
		return errors.New("error getting device: " + err.Error())
	}

	encryptionKey, err := m.getEncryptionKey(device)
	if err != nil {
		return errors.New("error extracting encryption key: " + err.Error())
	}

	// TEMPORARY ARRAY UNTIL WE HANDLE MULTI POINT WRITE
	points := []*model.Point{item.Point}
	deviceDescription := codec.GetDeviceDescription(device, codecs.LoRaDeviceDescriptions)

	payload, err := deviceDescription.EncodeRequestMessage(points)
	if err != nil {
		return errors.New("error encoding request: " + err.Error())
	}

	messageID := utils.GenerateRandomId()
	completePacket, err := aesutils.Encrypt(
		nstring.DerefString(item.Point.AddressUUID), // Note this is the device loraraw unique address
		payload,
		encryptionKey,
		utils.LORARAW_OPTS_REQUEST,
		messageID,
	)
	if err != nil {
		return errors.New("error encrypting data: " + err.Error())
	}

	queue.SetMessage(item, messageID, completePacket)
	return nil
}
