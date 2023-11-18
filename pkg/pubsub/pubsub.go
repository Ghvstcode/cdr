package pubsub

import (
	"errors"
	"sync"
)

// Pubsub struct represents a basic publish-subscribe system.
// It manages subscriptions and message broadcasting to channels.
type Pubsub struct {
	mu     sync.RWMutex
	subs   map[string]map[chan string]struct{}
	closed bool
}

// NewPubsub initializes and returns a new instance of Pubsub.
func NewPubsub() *Pubsub {
	return &Pubsub{
		subs:   make(map[string]map[chan string]struct{}),
		closed: false,
	}
}

// Subscribe adds a new subscriber channel for a specific topic.
// Returns an error if the Pubsub system is closed.
func (ps *Pubsub) Subscribe(topic string, ch chan string) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if ps.closed {
		return errors.New("pubsub system is closed")
	}

	if _, ok := ps.subs[topic]; !ok {
		ps.subs[topic] = make(map[chan string]struct{})
	}

	ps.subs[topic][ch] = struct{}{}
	return nil
}

// Unsubscribe removes a subscriber channel for a specific topic.
// Returns an error if the channel is not found in the topic.
func (ps *Pubsub) Unsubscribe(topic string, ch chan string) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if subscribers, ok := ps.subs[topic]; ok {
		if _, found := subscribers[ch]; found {
			delete(subscribers, ch)
			return nil
		}
		return errors.New("channel not found in topic")
	}
	return errors.New("topic not found")
}

// Publish broadcasts a message to all subscriber channels of a topic.
// Skips channels that are not ready to receive to prevent blocking.
// Returns an error if the Pubsub system is closed.
func (ps *Pubsub) Publish(topic string, msg string) error {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	if ps.closed {
		return errors.New("pubsub system is closed")
	}

	if subscribers, ok := ps.subs[topic]; ok {
		for ch := range subscribers {
			select {
			case ch <- msg:
				// Message sent successfully
			default:
				// Skip if the channel is not ready to receive
			}
		}
	}
	return nil
}

// Close terminates the Pubsub system and closes all subscriber channels.
// Returns an error if the Pubsub system is already closed.
func (ps *Pubsub) Close() error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if ps.closed {
		return errors.New("pubsub system is already closed")
	}

	ps.closed = true
	for _, subscribers := range ps.subs {
		for ch := range subscribers {
			close(ch)
		}
	}
	return nil
}
