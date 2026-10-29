package core

import "sync"

// stateWriter writes the state documents of one save outside the world,
// keeping only the newest one not written yet.
type stateWriter struct {
	write   func([]byte)
	mu      sync.Mutex
	pending []byte
	wake    chan struct{}
	quit    chan struct{}
	done    chan struct{}
}

func newStateWriter(write func([]byte)) *stateWriter {
	w := &stateWriter{write: write, wake: make(chan struct{}, 1), quit: make(chan struct{}), done: make(chan struct{})}
	go w.run()
	return w
}

func (w *stateWriter) put(b []byte) {
	w.mu.Lock()
	w.pending = b
	w.mu.Unlock()
	select {
	case w.wake <- struct{}{}:
	default:
	}
}

func (w *stateWriter) run() {
	defer close(w.done)
	for {
		select {
		case <-w.wake:
			w.mu.Lock()
			b := w.pending
			w.pending = nil
			w.mu.Unlock()
			if b != nil {
				w.write(b)
			}
		case <-w.quit:
			return
		}
	}
}

// stop ends the writer once the write in progress is done, dropping the
// pending state.
func (w *stateWriter) stop() {
	close(w.quit)
	<-w.done
}
