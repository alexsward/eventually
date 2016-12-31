package eventually

import (
	"errors"
	"time"
)

var (
	// ErrGetTimedOut when a call to Get(int, time.Duration) times out
	ErrGetTimedOut = errors.New("Get timed out")
	// ErrExecutorAlreadyStopped when the Executor is already stopped on a Stop() call
	ErrExecutorAlreadyStopped = errors.New("Executor already stopped")
)

// Executor is a service that will Execute Futures. It can be stopped and restarted
type Executor interface {
	// Execute queues the Future for execution and returns if it was successfully added
	Execute(Future) bool
	// Starts the Executor if it isn't already, returns whether this invocation started it or not
	Start() bool
	// Stop will cause any more Execute() calls to return false and allow no more Futures to be added
	Stop() error
	// Listen returns the output channel. Futures that complete (Future.Await() returns) will then be sent over this channel
	Listen() <-chan Future
	// Get will return total Futures or will timeout after duration and return up to total Futures
	Get(total int, duration time.Duration) ([]Future, error)
}

type executor struct {
	max              uint
	semaphore, kill  chan bool
	buffer           chan Future
	out              chan Future
	started, stopped bool
}

// NewExecutor creates a new Executor with the given size, optionally started, to execute Futures
func NewExecutor(size uint, start bool) Executor {
	e := &executor{
		max: size,
	}
	if start {
		e.Start()
	}
	return e
}

func (e *executor) Execute(f Future) bool {
	if e.stopped {
		return false
	}
	e.buffer <- f
	return true
}

func (e *executor) execute() {
	for {
		select {
		case <-e.kill:
			close(e.kill)
			return
		case f := <-e.buffer:
			go func(f Future) {
				e.semaphore <- true
				f.Await()
				<-e.semaphore
				e.out <- f
			}(f)
		}
	}
}

func (e *executor) Start() bool {
	if e.started {
		return false
	}
	e.semaphore = make(chan bool, e.max)
	e.kill = make(chan bool)
	e.buffer = make(chan Future)
	e.out = make(chan Future)
	e.started = true

	go e.execute()
	return true
}

func (e *executor) Stop() error {
	if e.stopped {
		return ErrExecutorAlreadyStopped
	}

	e.stopped = true
	e.kill <- true
	close(e.out)
	close(e.semaphore)
	close(e.buffer)
	return nil
}

func (e *executor) Listen() <-chan Future {
	return e.out
}

func (e *executor) Get(total int, duration time.Duration) ([]Future, error) {
	timeout := time.NewTicker(duration)
	defer timeout.Stop()
	var fs []Future
	for {
		select {
		case f := <-e.Listen():
			fs = append(fs, f)
			if len(fs) == total {
				return fs, nil
			}
		case <-timeout.C:
			return fs, ErrGetTimedOut
		}
	}
}
