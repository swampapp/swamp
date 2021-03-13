package status

import (
	"context"

	"github.com/swampapp/swamp/internal/eventbus"
)

const (
	SetRightEvent = "status.set_right"
	SetEvent      = "status.set"
	ErrorEvent    = "status.error"
)

func init() {
	eventbus.RegisterEvents(SetRightEvent, ErrorEvent, SetEvent)
}

func Error(text string) {
	eventbus.Emit(context.Background(), ErrorEvent, text)
}

func Set(text string) {
	eventbus.Emit(context.Background(), SetEvent, text)
}

func SetRight(text string) {
	eventbus.Emit(context.Background(), SetRightEvent, text)
}
