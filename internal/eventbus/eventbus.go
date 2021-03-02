package eventbus

import (
	"context"

	"github.com/mustafaturan/bus"
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

func ListenTo(topic string, handler func(evt *Event)) {
	h := &bus.Handler{
		Handle: func(e *bus.Event) {
			handler(&Event{e.Data})
		},
		Matcher: topic,
	}
	hid := xid.New().String()
	ebus.RegisterHandler(hid, h)
}
