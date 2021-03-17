package daemon

import "sync"

type SubnetAdded struct {
	lock     sync.RWMutex
	handlers []func(EventSubnetAdded)
}

func (ev *SubnetAdded) Subscribe(handler func(EventSubnetAdded)) {
	ev.lock.Lock()
	ev.handlers = append(ev.handlers, handler)
	ev.lock.Unlock()
}
func (ev *SubnetAdded) emit(payload EventSubnetAdded) {
	ev.lock.RLock()
	for _, handler := range ev.handlers {
		handler(payload)
	}
	ev.lock.RUnlock()
}

type SubnetRemoved struct {
	lock     sync.RWMutex
	handlers []func(EventSubnetRemoved)
}

func (ev *SubnetRemoved) Subscribe(handler func(EventSubnetRemoved)) {
	ev.lock.Lock()
	ev.handlers = append(ev.handlers, handler)
	ev.lock.Unlock()
}
func (ev *SubnetRemoved) emit(payload EventSubnetRemoved) {
	ev.lock.RLock()
	for _, handler := range ev.handlers {
		handler(payload)
	}
	ev.lock.RUnlock()
}

type Events struct {
	SubnetAdded   SubnetAdded
	SubnetRemoved SubnetRemoved
}

func (bus *Events) SubscribeAll(listener interface {
	SubnetAdded(payload EventSubnetAdded)
	SubnetRemoved(payload EventSubnetRemoved)
}) {
	bus.SubnetAdded.Subscribe(listener.SubnetAdded)
	bus.SubnetRemoved.Subscribe(listener.SubnetRemoved)
}
