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

type Ready struct {
	lock     sync.RWMutex
	handlers []func(EventReady)
}

func (ev *Ready) Subscribe(handler func(EventReady)) {
	ev.lock.Lock()
	ev.handlers = append(ev.handlers, handler)
	ev.lock.Unlock()
}
func (ev *Ready) emit() {
	payload := EventReady{}
	ev.lock.RLock()
	for _, handler := range ev.handlers {
		handler(payload)
	}
	ev.lock.RUnlock()
}

type Configured struct {
	lock     sync.RWMutex
	handlers []func(EventConfigured)
}

func (ev *Configured) Subscribe(handler func(EventConfigured)) {
	ev.lock.Lock()
	ev.handlers = append(ev.handlers, handler)
	ev.lock.Unlock()
}
func (ev *Configured) emit(payload EventConfigured) {
	ev.lock.RLock()
	for _, handler := range ev.handlers {
		handler(payload)
	}
	ev.lock.RUnlock()
}

type Events struct {
	SubnetAdded   SubnetAdded
	SubnetRemoved SubnetRemoved
	Ready         Ready
	Configured    Configured
}

func (bus *Events) SubscribeAll(listener interface {
	SubnetAdded(payload EventSubnetAdded)
	SubnetRemoved(payload EventSubnetRemoved)
	Ready(payload EventReady)
	Configured(payload EventConfigured)
}) {
	bus.SubnetAdded.Subscribe(listener.SubnetAdded)
	bus.SubnetRemoved.Subscribe(listener.SubnetRemoved)
	bus.Ready.Subscribe(listener.Ready)
	bus.Configured.Subscribe(listener.Configured)
}
