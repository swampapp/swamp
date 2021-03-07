package status

import (
	"context"

	"github.com/swampapp/swamp/internal/eventbus"
)

var SetRightEvent = "status.set_right"
var SetEvent = "status.set"
var ErrorEvent = "status.error"

func init() {
	eventbus.RegisterTopics(SetRightEvent, ErrorEvent, SetEvent)
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
