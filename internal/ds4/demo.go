package ds4

import (
	"sync"
	"time"
)

type demoRainbow struct {
	done chan struct{}
	once sync.Once
}

func newDemoRainbow() *demoRainbow {
	return &demoRainbow{done: make(chan struct{})}
}

func (r *demoRainbow) PID() int { return 4242 }

func (r *demoRainbow) Stop() error {
	r.once.Do(func() { close(r.done) })
	return nil
}

func (r *demoRainbow) Kill() error { return r.Stop() }

func (r *demoRainbow) Wait() error {
	<-r.done
	// Keep the demo bridge asynchronous so the UI exercises its real
	// stop/wait flow rather than completing inside Update.
	time.Sleep(40 * time.Millisecond)
	return nil
}

func (r *demoRainbow) Output() string { return "" }

func (r *demoRainbow) closed() bool {
	select {
	case <-r.done:
		return true
	default:
		return false
	}
}
