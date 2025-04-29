package channel

import (
	"sync"

	FbsNotification "github.com/jiyeyuran/mediasoup-go/internal/FBS/Notification"
)

type Handler func(event FbsNotification.Event, body *FbsNotification.BodyT)

type Msg struct {
	Data *FbsNotification.NotificationT
	next *Msg
}

// Subscription represents interest in a given subject.
type Subscription struct {
	mu   sync.Mutex
	sid  int64
	conn *Channel

	// run loop to process messages once the first message is received.
	once sync.Once

	// Subject that represents this subscription.
	Subject string

	delivered uint64
	max       uint64
	mcb       Handler
	closed    bool

	// Async linked list
	pHead *Msg
	pTail *Msg
	pCond *sync.Cond
}

// Unsubscribe will remove interest in the given subject.
func (s *Subscription) Unsubscribe() error {
	return s.AutoUnsubscribe(0)
}

// AutoUnsubscribe will issue an automatic Unsubscribe that is
// processed by the server when max messages have been received.
// This can be useful when sending a request to an unknown number
// of subscribers.
func (s *Subscription) AutoUnsubscribe(max int) error {
	if s == nil {
		return ErrBadSubscription
	}
	s.mu.Lock()
	conn := s.conn
	closed := s.closed
	s.mu.Unlock()
	if conn == nil || closed {
		return ErrBadSubscription
	}
	return conn.unsubscribe(s, max)
}
