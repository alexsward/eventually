package eventually

import (
	"errors"
	"time"
)

var (
	// ErrFutureTimedOut when the operation being performed times out
	ErrFutureTimedOut = errors.New("Future timed out")
)

// Operation is something to execute that will eventually complete
type Operation func() (interface{}, error)

// Future will perform an operation or timeout, and then provide the result of that operation
type Future interface {
	// Await will run and await the result of the operation
	Await() (interface{}, error)
	// Run will execute the Future's Operation
	Run()
	// Get will return the result of the Operation
	Get() (interface{}, error)
	// Kill will terminate the execution if it hasn't already completed, returns whether it was killed
	Kill() bool
}

// NewFuture returns a new Future for the Operation to timeout in the given amount of time
func NewFuture(op Operation, timeout time.Duration) Future {
	return &future{
		op:      op,
		timeout: timeout,
		done:    make(chan bool),
		kill:    make(chan bool),
	}
}

type future struct {
	op                          Operation
	timeout                     time.Duration
	completed, killed, timedout bool
	done, kill                  chan bool
	result                      interface{}
	err                         error
}

func (f *future) isDone() bool {
	return f.completed || f.timedout || f.killed
}

func (f *future) Kill() bool {
	if f.completed {
		return f.killed
	}
	f.kill <- true
	return true
}

func (f *future) Await() (interface{}, error) {
	f.Run()
	return f.Get()
}

func (f *future) Get() (interface{}, error) {
	if !f.completed {
		<-f.done
	}
	return f.result, f.err
}

func (f *future) Run() {
	go f.tick()
	go f.run()
}

func (f *future) tick() {
	tick := time.NewTicker(f.timeout)
	defer tick.Stop()
	select {
	case <-tick.C:
		if f.completed || f.killed {
			return
		}
		f.timedout = true
		f.err = ErrFutureTimedOut
		f.close()
	case <-f.kill:
		if f.isDone() {
			return
		}
		f.killed = true
		f.close()
	}
}

func (f *future) run() {
	defer f.close()
	result, err := f.op()
	if f.isDone() {
		return
	}
	f.result = result
	f.err = err
	f.completed = true
	f.done <- true
}

func (f *future) close() {
	close(f.kill)
	close(f.done)
}
