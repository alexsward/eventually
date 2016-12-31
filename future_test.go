package eventually

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

var (
	errExpected = errors.New("An error occured")
	ms10        = time.Millisecond * 10
	ms20        = time.Millisecond * 10
	ms50        = time.Millisecond * 50
)

func TestFutureDone(t *testing.T) {
	fmt.Println("-- TestFutureDone")
	tests := []struct {
		done                        bool
		completed, killed, timedout bool
	}{
		{true, true, false, false},
		{true, false, true, false},
		{true, false, false, true},
		{true, true, true, true},
		{false, false, false, false},
		{true, true, true, false},
		{true, false, true, true},
		{true, true, false, true},
	}
	for i, test := range tests {
		f := &future{
			completed: test.completed,
			killed:    test.killed,
			timedout:  test.timedout,
		}
		if f.isDone() != test.done {
			t.Errorf("Test %d failed: expected done:%t, got %t", i+1, test.done, !test.done)
		}
	}
}

func TestFutureRunGetAwait(t *testing.T) {
	fmt.Println("-- TestFutureRunGetAwait")
	tests := []struct {
		result                    interface{}
		err                       error
		success, failure, timeout time.Duration
		completed, timedout       bool
	}{
		{"result", nil, ms10, ms20, ms50, true, false},
		{nil, errExpected, ms50, ms10, ms50, true, false},
		{nil, ErrFutureTimedOut, ms50, ms50, ms10, false, true},
	}
	for i, test := range tests {
		for j := 0; j < 2; j++ {
			var r interface{}
			var err error
			op := testOp(test.result, test.err, test.success, test.failure)
			f := NewFuture(op, test.timeout)
			if i == 1 {
				f.Run()
				r, err = f.Get()
			} else {
				r, err = f.Await()
			}
			if err != test.err {
				t.Errorf("Test %d, run %d failed: expected err '%s', got '%s'", i+1, j+1, test.err, err)
				continue
			}
			if r != test.result {
				t.Errorf("Test %d, run %d failed: expected result '%s', got '%s'", i+1, j+1, test.result, r)
				continue
			}
			future, ok := f.(*future)
			if !ok {
				t.Errorf("Test %d, run %d failed: expected Future to be of type *future, got %T", i+i, j+1, f)
				continue
			}
			if future.completed != test.completed {
				t.Errorf("Test %d, run %d failed: expected completed future: %t, it was %t", i+1, j+1, test.completed, future.completed)
				continue
			}
			if future.killed {
				t.Errorf("Test %d, run %d failed: expected future to not be killed, it was", i+1, j+1)
				continue
			}
			if future.timedout != test.timedout {
				t.Errorf("Test %d, run %d failed: expected future to be timeout:%t, it was %t", i+1, j+1, test.timedout, future.timedout)
				continue
			}
		}
	}
}

func TestFutureTimeout(t *testing.T) {
	fmt.Println("-- TestFutureTimeout")
	f := NewFuture(testOp(nil, nil, ms10, ms10), time.Microsecond)
	r, err := f.Await()
	if r != nil {
		t.Errorf("Expected test to time out, got a result:%s", r)
	}
	if err != ErrFutureTimedOut {
		t.Errorf("Expected ErrFutureTimedOut, got %s", err)
		return
	}
	future, ok := f.(*future)
	if !ok {
		t.Errorf("Wasn't a *future...")
		return
	}
	if !future.timedout {
		t.Errorf("Expected future to be marked as timed out, it wasn't")
	}
}

func TestFutureKill(t *testing.T) {
	fmt.Println("-- TestFutureKill")
	f := NewFuture(testOp(nil, nil, ms50, ms50), time.Second)
	f.Run()
	killed := f.Kill()
	if !killed {
		t.Errorf("Future should have been killed")
		return
	}
	future, ok := f.(*future)
	if !ok {
		t.Fail()
		return
	}
	if !future.killed {
		t.Errorf("*future should have been killed")
	}
}

func testOp(result interface{}, err error, s, e time.Duration) Operation {
	return func() (interface{}, error) {
		successTick := time.NewTicker(s)
		failTick := time.NewTicker(e)
		timeoutTick := time.NewTicker(time.Second)
		defer successTick.Stop()
		defer failTick.Stop()
		defer timeoutTick.Stop()
		select {
		case <-successTick.C:
			return result, nil
		case <-failTick.C:
			return nil, err
		case <-timeoutTick.C:
			panic("Test panic'd, it timed out")
		}
	}
}
