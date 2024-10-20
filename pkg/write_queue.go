package pkg

import (
	"github.com/NubeIO/module-core-loraraw/endec"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

type PendingPointWrite struct {
	MessageId  uint8
	Point      *model.Point
	RetryCount int
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

func (pwq *PointWriteQueue) LoadWriteQueue(points []*model.Point) {
	pwq.mutex.Lock()
	defer pwq.mutex.Unlock()

	for _, point := range points {
		pendingPointWrite := &PendingPointWrite{Point: point}
		pwq.writeQueue = append(pwq.writeQueue, pendingPointWrite)
	}
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

		if pendingPointWrite.RetryCount <= pwq.maxRetry {
			pendingPointWrite.RetryCount++

			serialData := endec.NewSerialData()
			endec.SetPositionalData(serialData, true)
			endec.SetRequestData(serialData, true)
			msgId, _ := endec.GenerateRandomId()
			endec.SetMessageId(serialData, msgId)
			endec.UpdateBitPositionsForHeaderByte(serialData)

			pendingPointWrite.MessageId = msgId

			encryptionKey, err := getEncryptionKey(pendingPointWrite.Point.DeviceUUID)
			if err != nil {
				log.Errorf("error extracting encryption key: %s", err.Error())
				continue
			}

			encryptedData, err := endec.EncodeAndEncrypt(pendingPointWrite.Point, serialData, encryptionKey)
			if err != nil {
				log.Errorf("error encrypting data: %s", err.Error())
				continue
			}

			err = writeToLoRaRaw(encryptedData)
			if err != nil {
				log.Infof("error writing to LoRa: %v\n", err)
			}
		} else {
			pwq.DequeueWriteQueue()
		}

		// Wait for the set timeout before initiating another write
		time.Sleep(pwq.timeout)
	}
}
