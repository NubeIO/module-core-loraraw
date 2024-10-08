package pkg

import (
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

type PendingPointWrite struct {
	MessageId uint8
	Message   []byte
}

type PointWriteQueue struct {
	writeQueue []*PendingPointWrite
	mutex      sync.Mutex
	maxRetry   int
	timeout    time.Duration
}

func NewPointWriteQueue(maxRetry int, timeout time.Duration) *PointWriteQueue {
	queue := &PointWriteQueue{
		writeQueue: make([]*PendingPointWrite, 0),
		maxRetry:   maxRetry,
		timeout:    timeout,
	}
	return queue
}

func (pwq *PointWriteQueue) EnqueueWriteQueue(ppWrite *PendingPointWrite) {
	pwq.mutex.Lock()
	defer pwq.mutex.Unlock()

	pwq.writeQueue = append(pwq.writeQueue, ppWrite)
}

func (pwq *PointWriteQueue) DequeueWriteQueue() *PendingPointWrite {
	pwq.mutex.Lock()
	defer pwq.mutex.Unlock()

	if len(pwq.writeQueue) == 0 {
		return nil
	}

	queueItem := pwq.writeQueue[0]
	pwq.writeQueue = pwq.writeQueue[1:]
	return queueItem
}

func (pwq *PointWriteQueue) Size() int {
	pwq.mutex.Lock()
	defer pwq.mutex.Unlock()

	return len(pwq.writeQueue)
}

func (pwq *PointWriteQueue) ProcessPointWriteQueue(writeToLoRaRaw func([]byte) error) {
	for {
		pwq.mutex.Lock()

		if len(pwq.writeQueue) == 0 {
			pwq.mutex.Unlock()
			time.Sleep(time.Second * 5)
			continue
		}

		pendingPointWrite := pwq.writeQueue[0]
		pwq.mutex.Unlock()

		err := writeToLoRaRaw(pendingPointWrite.Message)
		if err != nil {
			log.Infof("Error writing to LoRa: %v\n", err)
			time.Sleep(pwq.timeout)
			continue
		}

		// Wait for the set timeout before initiating another write
		time.Sleep(pwq.timeout)
	}
}
