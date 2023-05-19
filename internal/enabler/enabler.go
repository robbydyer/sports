package enabler

import "go.uber.org/atomic"

type Enabler struct {
	enabled             *atomic.Bool
	stateChangeCallback func()
}

func New() *Enabler {
	return &Enabler{
		enabled: atomic.NewBool(false),
	}
}

func (e *Enabler) Enabled() bool {
	return e.enabled.Load()
}

func (e *Enabler) Enable() bool {
	return e.Store(true)
}

func (e *Enabler) Disable() bool {
	return e.Store(false)
}

func (e *Enabler) Store(set bool) bool {
	if e.enabled.CompareAndSwap(!set, set) {
		if e.stateChangeCallback != nil {
			e.stateChangeCallback()
		}
		return true
	}
	return false
}

func (e *Enabler) SetStateChangeCallback(s func()) {
	e.stateChangeCallback = s
}
