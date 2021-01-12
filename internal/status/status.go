package status

import "fmt"

func Error(text string) {
	if onError != nil {
		onError(fmt.Sprintf("🛑 %s", text))
	}
}

func Set(text string) {
	if onSet != nil {
		onSet(text)
	}
}

func SetRight(text string) {
	if onSetRight != nil {
		onSetRight(text)
	}
}

var onSetRight func(string)

func OnSetRight(fn func(string)) {
	onSetRight = fn
}

var onError func(string)

func OnError(fn func(string)) {
	onError = fn
}

var onSet func(string)

func OnSet(fn func(string)) {
	onSet = fn
}
