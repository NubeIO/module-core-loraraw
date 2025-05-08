package pkg

import (
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

type PendingPointWrite struct {
	MessageId   uint8
	Message     []byte
	MessageType bool
	Point       *model.Point
	RetryCount  int
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
	pendingPointWrite := &PendingPointWrite{Point: point}
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
	getDevice func(string) (*model.Device, error),
	getEncryptionKey func(*model.Device) ([]byte, error),
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
			device, err := getDevice(pendingPointWrite.Point.DeviceUUID)
			if err != nil {
				log.Errorf("error getting device: %s", err.Error())
				continue
			}

			encryptionKey, err := getEncryptionKey(device)
			if err != nil {
				log.Errorf("error extracting encryption key: %s", err.Error())
				continue
			}

			// TEMPORARY ARRAY UNTIL WE HANDLE MULTI POINT WRITE
			points := []*model.Point{pendingPointWrite.Point}
			deviceDescription := codec.GetDeviceDescription(device, codecs.LoRaDeviceDescriptions)

			payload, err := deviceDescription.EncodeRequestMessage(points)

			messageID := utils.GenerateRandomId()
			completePacket, err := aesutils.Encrypt(
				nstring.DerefString(pendingPointWrite.Point.AddressUUID), // Note this is the device loraraw unique address
				payload,
				encryptionKey,
				utils.LORARAW_OPTS_REQUEST,
				messageID,
			)

			if err != nil {
				log.Errorf("error encrypting data: %s", err.Error())
				// Removing the point from the queue as queued point may be invalid
				pwq.DequeueWriteQueue()
				continue
			}

			pendingPointWrite.MessageId = messageID
			pendingPointWrite.Message = completePacket

		}

		if pendingPointWrite.RetryCount < pwq.maxRetry {
			err := writeToLoRaRaw(pendingPointWrite.Message)
			if err != nil {
				log.Errorf("error writing to LoRa serial port: %v\n", err)
				time.Sleep(time.Second * 2)
				continue
			}
			pendingPointWrite.RetryCount++

			// Wait for the set timeout before initiating another write
			time.Sleep(pwq.timeout)
		} else {
			pwq.DequeueWriteQueue()
		}
	}
}
