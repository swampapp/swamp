package observer

import (
	"sync"
)

type (
	Observable interface {
		Add(observer Observer)
		Notify(event string)
		Remove(observer Observer)
	}

	Observer interface {
		NotifyCallback(event string)
	}

	BasicObservable struct {
		observer sync.Map
	}
)

func New() Observable {
	return &BasicObservable{observer: sync.Map{}}
}
func (wt *BasicObservable) Add(observer Observer) {
	wt.observer.Store(observer, struct{}{})
}

func (wt *BasicObservable) Remove(observer Observer) {
	wt.observer.Delete(observer)
}

func (wt *BasicObservable) Notify(event string) {
	wt.observer.Range(func(key, value interface{}) bool {
		if key == nil {
			return false
		}

		key.(Observer).NotifyCallback(event)
		return true
	})
}
