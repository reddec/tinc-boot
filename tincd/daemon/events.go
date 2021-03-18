package daemon

import "sync"

type Configured struct {
	lock     sync.RWMutex
	handlers []func(Configuration)
}

func (ev *Configured) Subscribe(handler func(Configuration)) {
	ev.lock.Lock()
	ev.handlers = append(ev.handlers, handler)
	ev.lock.Unlock()
}
func (ev *Configured) emit(payload Configuration) {
	ev.lock.RLock()
	for _, handler := range ev.handlers {
		handler(payload)
	}
	ev.lock.RUnlock()
}

type Stopped struct {
	lock     sync.RWMutex
	handlers []func(Configuration)
}

func (ev *Stopped) Subscribe(handler func(Configuration)) {
	ev.lock.Lock()
	ev.handlers = append(ev.handlers, handler)
	ev.lock.Unlock()
}
func (ev *Stopped) emit(payload Configuration) {
	ev.lock.RLock()
	for _, handler := range ev.handlers {
		handler(payload)
	}
	ev.lock.RUnlock()
}

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

type Events struct {
	Configured    Configured
	Stopped       Stopped
	SubnetAdded   SubnetAdded
	SubnetRemoved SubnetRemoved
	Ready         Ready
}

func (bus *Events) SubscribeAll(listener interface {
	Configured(payload Configuration)
	Stopped(payload Configuration)
	SubnetAdded(payload EventSubnetAdded)
	SubnetRemoved(payload EventSubnetRemoved)
	Ready(payload EventReady)
}) {
	bus.Configured.Subscribe(listener.Configured)
	bus.Stopped.Subscribe(listener.Stopped)
	bus.SubnetAdded.Subscribe(listener.SubnetAdded)
	bus.SubnetRemoved.Subscribe(listener.SubnetRemoved)
	bus.Ready.Subscribe(listener.Ready)
}
