package pkg

import (
	"sync"

	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
	log "github.com/sirupsen/logrus"
)

type PendingPointWrite struct {
	MessageId   uint8
	Message     []byte
	MessageType bool
	Point       *model.Point
	RetryCount  int

	// done is closed exactly once when the item leaves the queue (acked,
	// exhausted or dropped). The scheduler waits on it after transmitting.
	done     chan struct{}
	doneOnce sync.Once
	acked    bool
}

func (p *PendingPointWrite) markDone(acked bool) {
	p.doneOnce.Do(func() {
		p.acked = acked
		close(p.done)
	})
}

// --------------------------------------------
// SINGLE QUEUE — holds one device's pending writes in order.
// Transmission is driven by PointWriteQueueManager's scheduler.
// --------------------------------------------

type PointWriteQueue struct {
	writeQueue []*PendingPointWrite
	mutex      sync.Mutex
}

func NewPointWriteQueue() *PointWriteQueue {
	return &PointWriteQueue{
		writeQueue: make([]*PendingPointWrite, 0),
	}
}

func (pwq *PointWriteQueue) EnqueueWriteQueue(point *model.Point) *PendingPointWrite {
	pwq.mutex.Lock()
	defer pwq.mutex.Unlock()

	ppWrite := &PendingPointWrite{Point: point, done: make(chan struct{})}
	pwq.writeQueue = append(pwq.writeQueue, ppWrite)
	return ppWrite
}

// Peek returns the head item without removing it, or nil when empty.
func (pwq *PointWriteQueue) Peek() *PendingPointWrite {
	pwq.mutex.Lock()
	defer pwq.mutex.Unlock()

	if len(pwq.writeQueue) == 0 {
		return nil
	}
	return pwq.writeQueue[0]
}

// SetMessage stores the encoded frame on the item under the queue lock so the
// RX side never observes a half-initialised MessageId.
func (pwq *PointWriteQueue) SetMessage(item *PendingPointWrite, messageId uint8, message []byte) {
	pwq.mutex.Lock()
	defer pwq.mutex.Unlock()

	item.MessageId = messageId
	item.Message = message
}

// IncRetry bumps the attempt counter and returns the new value.
func (pwq *PointWriteQueue) IncRetry(item *PendingPointWrite) int {
	pwq.mutex.Lock()
	defer pwq.mutex.Unlock()

	item.RetryCount++
	return item.RetryCount
}

// RemoveItem removes the given item if it is still the head of the queue.
func (pwq *PointWriteQueue) RemoveItem(item *PendingPointWrite) bool {
	pwq.mutex.Lock()
	defer pwq.mutex.Unlock()

	if len(pwq.writeQueue) == 0 || pwq.writeQueue[0] != item {
		return false
	}
	pwq.writeQueue = pwq.writeQueue[1:]
	item.markDone(false)
	return true
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

// dequeue removes the head. With a messageId it only removes the head when the
// id matches (that is the ack path) and marks the item as acked.
func (pwq *PointWriteQueue) dequeue(messageId *uint8) *PendingPointWrite {
	if len(pwq.writeQueue) == 0 {
		return nil
	}

	head := pwq.writeQueue[0]
	if messageId == nil {
		pwq.writeQueue = pwq.writeQueue[1:]
		head.markDone(false)
		return head
	}

	if head.Message == nil || head.MessageId != *messageId {
		return nil
	}
	pwq.writeQueue = pwq.writeQueue[1:]
	head.markDone(true)
	return head
}

func (pwq *PointWriteQueue) Size() int {
	pwq.mutex.Lock()
	defer pwq.mutex.Unlock()

	return len(pwq.writeQueue)
}
