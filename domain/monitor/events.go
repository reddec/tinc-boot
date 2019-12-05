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

type eventStarted struct {
	lock     sync.RWMutex
	handlers []func(*service)
}

func (ev *eventStarted) Subscribe(handler func(*service)) {
	ev.lock.Lock()
	ev.handlers = append(ev.handlers, handler)
	ev.lock.Unlock()
}
func (ev *eventStarted) emit(payload *service) {
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
	Started      eventStarted
}

func (bus *monitorEvents) SubscribeAll(listener interface {
	Connected(payload *Node)
	Disconnected(payload *Node)
	Fetched(payload *Node)
	Started(payload *service)
}) {
	bus.Connected.Subscribe(listener.Connected)
	bus.Disconnected.Subscribe(listener.Disconnected)
	bus.Fetched.Subscribe(listener.Fetched)
	bus.Started.Subscribe(listener.Started)
}
