package pkg

import (
	"github.com/NubeIO/module-core-loraraw/endec"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

type PointWriteState string

const (
	PointWritePending PointWriteState = "point-write-pending"
	PointWriteSuccess PointWriteState = "point-write-success"
)

type PendingPointWrite struct {
	MessageId        uint8
	Message          []byte
	Point            *model.Point
	RetryCount       int
	PointWriteStatus PointWriteState
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

func (pwq *PointWriteQueue) LoadWriteQueue(point *model.Point) {
	pwq.mutex.Lock()
	defer pwq.mutex.Unlock()
	pendingPointWrite := &PendingPointWrite{Point: point, PointWriteStatus: PointWritePending}
	pwq.writeQueue = append(pwq.writeQueue, pendingPointWrite)
}

func (pwq *PointWriteQueue) EnqueueWriteQueue(ppWrite *PendingPointWrite) {
	pwq.mutex.Lock()
	defer pwq.mutex.Unlock()

	pwq.writeQueue = append(pwq.writeQueue, ppWrite)
}

func (pwq *PointWriteQueue) DequeueWriteQueue() {
	pwq.mutex.Lock()
	defer pwq.mutex.Unlock()

	pwq.dequeue(nil)
}

func (pwq *PointWriteQueue) DequeueUsingMessageId(messageId uint8) *model.Point {
	pwq.mutex.Lock()
	defer pwq.mutex.Unlock()

	pendingPointWrite := pwq.dequeue(&messageId)
	if pendingPointWrite == nil {
		log.Errorf("no pending point write found for messageId %v", messageId)
		return nil
	}
	return pendingPointWrite.Point
}

func (pwq *PointWriteQueue) dequeue(messageId *uint8) *PendingPointWrite {
	if len(pwq.writeQueue) == 0 {
		return nil
	}

	var dequeuedItem *PendingPointWrite

	if messageId == nil {
		dequeuedItem = pwq.writeQueue[0]
		pwq.writeQueue = pwq.writeQueue[1:]
	} else {
		queueItem := pwq.writeQueue[0]
		if queueItem.MessageId == *messageId {
			dequeuedItem = pwq.writeQueue[0]
			pwq.writeQueue = pwq.writeQueue[1:]
		}
	}

	return dequeuedItem
}

func (pwq *PointWriteQueue) Size() int {
	pwq.mutex.Lock()
	defer pwq.mutex.Unlock()

	return len(pwq.writeQueue)
}

func (pwq *PointWriteQueue) ProcessPointWriteQueue(
	getEncryptionKey func(string) ([]byte, error),
	writeToLoRaRaw func([]byte) error,
) {
	for {
		pwq.mutex.Lock()

		if len(pwq.writeQueue) == 0 {
			pwq.mutex.Unlock()
			time.Sleep(time.Second * 5)
			continue
		}

		pendingPointWrite := pwq.writeQueue[0]
		pwq.mutex.Unlock()

		if pendingPointWrite.Message == nil {
			serialData := endec.NewSerialData()
			endec.SetPositionalData(serialData, true)
			endec.SetRequestData(serialData, true)
			msgId, _ := endec.GenerateRandomId()
			endec.SetMessageId(serialData, msgId)
			endec.UpdateBitPositionsForHeaderByte(serialData)

			encryptionKey, err := getEncryptionKey(pendingPointWrite.Point.DeviceUUID)
			if err != nil {
				log.Errorf("error extracting encryption key: %s", err.Error())
				continue
			}

			encryptedData, err := endec.EncodeAndEncrypt(pendingPointWrite.Point, serialData, encryptionKey)
			if err != nil {
				log.Errorf("error encrypting data: %s", err.Error())
				// Removing the point from the queue as queued point may be invalid
				pwq.DequeueWriteQueue()
				continue
			}

			pendingPointWrite.MessageId = msgId
			pendingPointWrite.Message = encryptedData
		}

		if pendingPointWrite.RetryCount < pwq.maxRetry {
			if pendingPointWrite.PointWriteStatus == PointWritePending {
				err := writeToLoRaRaw(pendingPointWrite.Message)
				if err != nil {
					log.Infof("error writing to LoRa: %v\n", err)
					pendingPointWrite.RetryCount++
					continue
				}
				pendingPointWrite.PointWriteStatus = PointWriteSuccess
			}
		} else {
			pwq.DequeueWriteQueue()
		}

		// Wait for the set timeout before initiating another write
		time.Sleep(pwq.timeout)
	}
}
