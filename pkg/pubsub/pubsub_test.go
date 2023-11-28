package pubsub

import (
	"sync"
	"testing"
	"time"
)

func TestNewPubsub(t *testing.T) {
	ps := NewPubsub()
	if ps == nil {
		t.Errorf("NewPubsub() = nil, want non-nil")
	}
}

func TestSubscribeAndPublish(t *testing.T) {
	ps := NewPubsub()
	topic := "testTopic"
	ch := make(chan string, 1)

	if err := ps.Subscribe(topic, ch); err != nil {
		t.Errorf("Subscribe() error = %v, wantErr false", err)
	}

	msg := "hello"
	if err := ps.Publish(topic, msg); err != nil {
		t.Errorf("Publish() error = %v, wantErr false", err)
	}

	receivedMsg := <-ch
	if receivedMsg != msg {
		t.Errorf("Publish() got = %v, want %v", receivedMsg, msg)
	}
}

func TestUnsubscribe(t *testing.T) {
	ps := NewPubsub()
	topic := "testTopic"
	ch := make(chan string, 1)

	_ = ps.Subscribe(topic, ch)

	if err := ps.Unsubscribe(topic, ch); err != nil {
		t.Errorf("Unsubscribe() error = %v, wantErr false", err)
	}

	if err := ps.Publish(topic, "message"); err != nil {
		t.Errorf("Publish() after unsubscribe error = %v, wantErr false", err)
	}

	select {
	case <-ch:
		t.Errorf("Received message after unsubscribe")
	case <-time.After(50 * time.Millisecond):
		// No message received as expected
	}
}

func TestClose(t *testing.T) {
	ps := NewPubsub()
	ch := make(chan string, 1)

	_ = ps.Subscribe("testTopic", ch)

	if err := ps.Close(); err != nil {
		t.Errorf("Close() error = %v, wantErr false", err)
	}

	if _, ok := <-ch; ok {
		t.Errorf("Channel should be closed")
	}
}

func TestPublishToClosedPubsub(t *testing.T) {
	ps := NewPubsub()
	_ = ps.Close()

	if err := ps.Publish("testTopic", "message"); err == nil {
		t.Errorf("Publish() to closed pubsub got no error, wantErr true")
	}
}

func TestSubscribeToClosedPubsub(t *testing.T) {
	ps := NewPubsub()
	_ = ps.Close()

	if err := ps.Subscribe("testTopic", make(chan string, 1)); err == nil {
		t.Errorf("Subscribe() to closed pubsub got no error, wantErr true")
	}
}

func TestConcurrency(t *testing.T) {
	ps := NewPubsub()
	topic := "testTopic"
	numSubscribers := 10
	var wg sync.WaitGroup

	// Creating multiple subscribers
	for i := 0; i < numSubscribers; i++ {
		ch := make(chan string, 1)
		_ = ps.Subscribe(topic, ch)

		wg.Add(1)
		go func() {
			defer wg.Done()
			if msg := <-ch; msg != "message" {
				t.Errorf("Received wrong message: got %v, want 'message'", msg)
			}
		}()
	}

	// Publishing a message
	_ = ps.Publish(topic, "message")

	wg.Wait()
}
