package deadlock

import (
	"testing"
	"time"
)

var (
	deadlock     bool
	deadlockType int
	infos        []DeadlockInfo
)

func getOnDeadlockFunc() OnDeadlockFunc {
	deadlock = false
	deadlockType = 0
	infos = infos[:0]
	return func(deadlockType_ int, infos_ []DeadlockInfo) {
		deadlock = true
		deadlockType = deadlockType_
		infos = infos_
	}
}

func TestDeadlock(t *testing.T) {
	var tds = []struct {
		no           int
		f            func()
		deadlock     bool
		deadlockType int
	}{
		{
			no:           1,
			f:            normalCase,
			deadlock:     false,
			deadlockType: 0,
		},
		{
			no:           2,
			f:            deadlockCase1,
			deadlock:     true,
			deadlockType: TypeReLock,
		},
		{
			no:           3,
			f:            deadlockCase2,
			deadlock:     true,
			deadlockType: TypeCyclic,
		},
		{
			no:           4,
			f:            deadlockCase3,
			deadlock:     true,
			deadlockType: TypeCyclic,
		},
		{
			no:           5,
			f:            deadlockCase4,
			deadlock:     true,
			deadlockType: TypeCyclic,
		},
	}

	for i, td := range tds {
		enableCheckDeadlock()
		td.f()
		if deadlock != td.deadlock {
			t.Errorf("td(%d): expect deadlock to be %v, actual %v", i+1, td.deadlock, deadlock)
		}

		if deadlockType != td.deadlockType {
			t.Errorf("td(%d): expect deadlockType to be %v, actual %v", i+1, td.deadlockType, deadlockType)
		}

		if deadlock {
			t.Log(DefaultDeadlockInfoString(deadlockType, infos))
		}
	}
}

func enableCheckDeadlock() {
	SetOptions(Options{
		CheckDeadLock: true,
		OnDeadlock:    getOnDeadlockFunc(),
	})
}

func normalCase() {
	var a, b Mutex
	go func() {
		a.Lock()
		b.Lock()
		a.Unlock()
		b.Unlock()
	}()

	go func() {
		a.Lock()
		a.Unlock()
	}()

	time.Sleep(1 * time.Second)
}

func deadlockCase1() {
	var a Mutex
	go func() {
		a.Lock()
		defer a.Unlock()
		func() {
			a.Lock()
			defer a.Unlock()
		}()
	}()
	time.Sleep(time.Second)
}

func deadlockCase2() {
	var a, b, c Mutex
	go func() {
		a.Lock()
		b.Lock()
		c.Lock()
		a.Unlock()
		b.Unlock()
		c.Unlock()
	}()
	time.Sleep(100 * time.Millisecond)
	go func() {
		c.Lock()
		a.Lock()
		a.Unlock()
		c.Unlock()
	}()
	time.Sleep(time.Second)
}

func deadlockCase3() {
	var a, b, c, d, e RWMutex
	go func() {
		a.RLock()
		b.RLock()
		c.RLock()
		d.RLock()
		e.RLock()
		a.RUnlock()
		b.RUnlock()
		c.RUnlock()
		d.RUnlock()
		e.RUnlock()
	}()
	time.Sleep(time.Second)
	go func() {
		e.Lock()
		d.Lock()
		d.Unlock()
		e.Unlock()
	}()
	time.Sleep(time.Second)
}

func deadlockCase4() {
	var a, b, c, d, e RWMutex
	go func() {
		a.Lock()
		b.Lock()
		a.Unlock()
		b.Unlock()
	}()
	time.Sleep(100 * time.Millisecond)
	go func() {
		b.Lock()
		c.Lock()
		b.Unlock()
		c.Unlock()
	}()
	time.Sleep(100 * time.Millisecond)
	go func() {
		c.Lock()
		d.Lock()
		c.Unlock()
		d.Unlock()
	}()
	time.Sleep(100 * time.Millisecond)
	go func() {
		d.Lock()
		e.Lock()
		d.Unlock()
		e.Unlock()
	}()
	time.Sleep(100 * time.Millisecond)
	go func() {
		e.Lock()
		a.Lock()
		e.Unlock()
		a.Unlock()
	}()
	time.Sleep(100 * time.Millisecond)
}
