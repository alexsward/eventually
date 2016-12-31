package eventually

import (
	"fmt"
	"testing"
	"time"
)

func TestExecutorStart(t *testing.T) {
	fmt.Println("-- TestExecutorStart")
	e := NewExecutor(1, false)
	exec, ok := e.(*executor)
	if !ok {
		t.FailNow()
	}
	if exec.started {
		t.Errorf("Executor should not be started")
	}
	started := e.Start()
	if !exec.started {
		t.Errorf("Executor should be started")
	}
	if !started {
		t.Errorf("First Start() invocation should have started it")
	}
	restarted := e.Start()
	if restarted {
		t.Errorf("This executor was already started, Start() should be false")
	}
}

func TestExecutorStop(t *testing.T) {
	fmt.Println("-- TestExecutorStop")
	e := NewExecutor(1, true)
	err := e.Stop()
	if err != nil {
		t.Errorf("Err stopping first time should be nil, got: %s", err)
		t.FailNow()
	}
	err = e.Stop()
	if err != ErrExecutorAlreadyStopped {
		t.Errorf("Expected error: %s, got %s", ErrExecutorAlreadyStopped, err)
		t.FailNow()
	}
	submitted := e.Execute(NewFuture(testOp("1", nil, ms10, ms50), time.Second))
	if submitted {
		t.Errorf("Stopped executor should not accept future")
	}
}

func TestExecutorGet(t *testing.T) {
	fmt.Println("-- TestExecutorGet")
	tests := []struct {
		add, get int
		timeout  time.Duration
		err      error
	}{
		{1, 1, time.Second, nil},
		{2, 1, time.Second, nil},
		{10, 1, time.Second, nil},
		{10, 4, time.Second, nil},
		{1, 1, time.Microsecond, ErrGetTimedOut},
		{1, 10, time.Microsecond, ErrGetTimedOut},
	}
	for i, test := range tests {
		e := NewExecutor(4, true)
		for j := 0; j < test.add; j++ {
			e.Execute(NewFuture(testOp(fmt.Sprintf("%d", j), nil, ms10, ms50), time.Second))
		}
		fs, err := e.Get(test.get, test.timeout)
		if err != test.err {
			t.Errorf("Test %d failed: expected error '%s', got '%s'", i+1, test.err, err)
			continue
		}
		if test.err != nil {
			continue
		}
		if len(fs) != test.get {
			t.Errorf("Test %d failed: expected %d items, got %d", i+1, test.get, len(fs))
			continue
		}
	}
}
