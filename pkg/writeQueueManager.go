package pkg

import (
	"sync"
	"time"

	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
	log "github.com/sirupsen/logrus"
)

// ---------------------------------------------------
// MANAGER — manages multiple queues by DeviceUUID
// ---------------------------------------------------

type PointWriteQueueManager struct {
	queues            map[string]*PointWriteQueue
	mutex             sync.Mutex
	maxRetry          int
	timeOffAirDefault time.Duration

	getDevice        func(string) (*model.Device, error)
	getEncryptionKey func(*model.Device) ([]byte, error)
	writeToLoRaRaw   func([]byte) error
}

func NewPointWriteQueueManager(
	maxRetry int,
	defaultTimeOffAir time.Duration,
	getDevice func(string) (*model.Device, error),
	getEncryptionKey func(*model.Device) ([]byte, error),
	writeToLoRaRaw func([]byte) error,
) *PointWriteQueueManager {
	return &PointWriteQueueManager{
		queues:            make(map[string]*PointWriteQueue),
		maxRetry:          maxRetry,
		timeOffAirDefault: defaultTimeOffAir,
		getDevice:         getDevice,
		getEncryptionKey:  getEncryptionKey,
		writeToLoRaRaw:    writeToLoRaRaw,
	}
}

func (m *PointWriteQueueManager) getOrCreateQueue(deviceUUID string) *PointWriteQueue {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	queue, exists := m.queues[deviceUUID]
	if !exists {
		queue = NewPointWriteQueue(m.maxRetry, m.timeOffAirDefault)
		m.queues[deviceUUID] = queue
		go queue.ProcessPointWriteQueue(m.getDevice, m.getEncryptionKey, m.writeToLoRaRaw)
		log.Infof("created new write queue for device %s", deviceUUID)
	}
	return queue
}

func (m *PointWriteQueueManager) EnqueuePoint(point *model.Point) {
	queue := m.getOrCreateQueue(point.DeviceUUID)
	queue.EnqueueWriteQueue(point)
}

func (m *PointWriteQueueManager) DequeueUsingMessageId(deviceUUID string, messageId uint8) *model.Point {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	queue, exists := m.queues[deviceUUID]
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
