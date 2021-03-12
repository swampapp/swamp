// Package eventbus is a thin wrapper around mustafaturan/bus to provide
// a singleton event bus swamp can use.
package eventbus

import (
	"context"

	"github.com/mustafaturan/bus/v2"
	"github.com/rs/xid"
	"github.com/swampapp/swamp/internal/logger"
)

var ebus *bus.Bus

type generator struct {
}

func (g generator) Generate() string {
	return xid.New().String()
}

type Event struct {
	Data interface{}
}

func init() {
	var err error
	ebus, err = bus.NewBus(generator{})
	if err != nil {
		panic(err)
	}
}

func RegisterTopics(topics ...string) {
	ebus.RegisterTopics(topics...)
}

func Emit(ctx context.Context, topic string, data interface{}) {
	_, err := ebus.Emit(ctx, topic, data)
	if err != nil {
		logger.Errorf(err, "error emitting event for %s", topic)
	}
}

// Thin wrapper around bus.RegisterHandler that takes care of
// generating a unique handler ID for each listener
func ListenTo(topic string, handler func(evt *Event)) {
	h := &bus.Handler{
		Handle: func(ctx context.Context, e *bus.Event) {
			handler(&Event{e.Data})
		},
		Matcher: topic,
	}
	hid := xid.New().String()
	ebus.RegisterHandler(hid, h)
}
