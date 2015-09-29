package bake

/*
// Event represents an event that occurs on a target.
type Event interface {
	event()
}

func (*CreatePackageEvent) event() {}

// CreatePackageEvent represents the creation of a package on a workspace.
type CreatePackageEvent struct {
	Package *Package
}

func (*ReadFileEvent) event() {}

// ReadFileEvent represents a read of a file by a target.
type ReadFileEvent struct {
	Target Target
	File   *File
}

// WriteFileEvent represents a write of a file by a target.
type WriteFileEvent struct {
	Target Target
	File   *File
}

// EventHandler represents an object that can receive events.
type EventHandler interface {
	HandleEvent(Event)
}

// EventHandlerFunc is a adapter to allow functions to implement EventHandler.
type EventHandlerFunc func(Event)

// HandleEvent calls fn(e).
func (fn EventHandlerFunc) HandleEvent(e Event) { fn(e) }

// EventDispatcher represents an object that can register handlers and dispatch events.
type EventDispatcher interface {
	AddHandler(h EventHandler)
	RemoveHandler(h EventHandler)
}

// dispatcher manages event handlers and dispatches events.
type dispatcher struct {
	handlers []EventHandler
}

// newDispatcher returns a new instance of dispatcher.
func newDispatcher() *dispatcher {
	return &dispatcher{}
}

// addHandler adds h to the set of event listeners.
func (d *dispatcher) addHandler(h EventHandler) {
	d.handlers = append(d.handlers, h)
}

// removeHandler deletes h from the set of event listeners.
func (d *dispatcher) removeHandler(h EventHandler) {
	for i, handler := range d.handlers {
		if handler == h {
			copy(d.handlers[i:], d.handlers[i+1:])
			d.handlers[len(d.handlers)-1] = nil
			d.handlers = d.handlers[:len(d.handlers)-1]
		}
	}
}

// dispatch sends e to all registered handlers.
func (d *dispatcher) dispatch(e Event) {
	for _, h := range d.handlers {
		h.HandleEvent(e)
	}
}

type handlerFuncEntry struct {
	EventHandlerFunc
}

func (h handlerFuncEntry) HandleEvent(e Event) { h.EventHandlerFunc(e) }
*/
