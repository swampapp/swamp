package eventbus

import (
	"context"
	"testing"
)

// Test we can add mulitple listeners for a topic
func TestListenMultiple(t *testing.T) {
	topic := "foobar"

	RegisterTopics(topic)

	l1, l2 := false, false

	ListenTo(topic, func(evt *Event) {
		l1 = true
	})

	ListenTo(topic, func(evt *Event) {
		l2 = true
	})

	Emit(context.Background(), topic, nil)

	if !l1 || !l2 {
		t.Error("both listeners should have received the event")
	}
}
