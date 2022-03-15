package porter

type Dispatcher struct {
	onRunHandlers      errorHandlers
	onShutdownHandlers errorHandlers
}

type Subscriber interface {
	ListenRun(handlers ...func(error))
	ListenShutdown(handlers ...func(error))
}

func (d *Dispatcher) OnRun(err error) {
	d.onRunHandlers.Invoke(err)
}

func (d *Dispatcher) OnShutdown(err error) {
	d.onShutdownHandlers.Invoke(err)
}

func (d *Dispatcher) ListenRun(handlers ...func(error)) {
	d.onRunHandlers = append(d.onRunHandlers, handlers...)
}

func (d *Dispatcher) ListenShutdown(handlers ...func(error)) {
	d.onShutdownHandlers = append(d.onShutdownHandlers, handlers...)
}

type errorHandlers []func(error)

func (h errorHandlers) Invoke(err error) {
	for _, handler := range h {
		handler(err)
	}
}
