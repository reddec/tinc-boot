package monitor

import "sync"

type eventConnected struct {
	lock     sync.RWMutex
	handlers []func(*Node)
}

func (ev *eventConnected) Subscribe(handler func(*Node)) {
	ev.lock.Lock()
	ev.handlers = append(ev.handlers, handler)
	ev.lock.Unlock()
}
func (ev *eventConnected) emit(payload *Node) {
	ev.lock.RLock()
	for _, handler := range ev.handlers {
		handler(payload)
	}
	ev.lock.RUnlock()
}

type eventDisconnected struct {
	lock     sync.RWMutex
	handlers []func(*Node)
}

func (ev *eventDisconnected) Subscribe(handler func(*Node)) {
	ev.lock.Lock()
	ev.handlers = append(ev.handlers, handler)
	ev.lock.Unlock()
}
func (ev *eventDisconnected) emit(payload *Node) {
	ev.lock.RLock()
	for _, handler := range ev.handlers {
		handler(payload)
	}
	ev.lock.RUnlock()
}

type eventFetched struct {
	lock     sync.RWMutex
	handlers []func(*Node)
}

func (ev *eventFetched) Subscribe(handler func(*Node)) {
	ev.lock.Lock()
	ev.handlers = append(ev.handlers, handler)
	ev.lock.Unlock()
}
func (ev *eventFetched) emit(payload *Node) {
	ev.lock.RLock()
	for _, handler := range ev.handlers {
		handler(payload)
	}
	ev.lock.RUnlock()
}

type monitorEvents struct {
	Connected    eventConnected
	Disconnected eventDisconnected
	Fetched      eventFetched
}

func (bus *monitorEvents) Sink(sink func(eventName string, payload interface{})) *monitorEvents {
	bus.Connected.Subscribe(func(payload *Node) {
		sink("Connected", payload)
	})
	bus.Disconnected.Subscribe(func(payload *Node) {
		sink("Disconnected", payload)
	})
	bus.Fetched.Subscribe(func(payload *Node) {
		sink("Fetched", payload)
	})
	return bus
}
func (bus *monitorEvents) SubscribeAll(listener interface {
	Connected(payload *Node)
	Disconnected(payload *Node)
	Fetched(payload *Node)
}) {
	bus.Connected.Subscribe(listener.Connected)
	bus.Disconnected.Subscribe(listener.Disconnected)
	bus.Fetched.Subscribe(listener.Fetched)
}
