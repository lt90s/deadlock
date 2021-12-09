package deadlock

import (
	"sync"
	"unsafe"
)

type Mutex struct {
	sync.Mutex
}

func (l *Mutex) Lock() {
	if opts.CheckDeadLock {
		addLockOp(unsafe.Pointer(l), lockOpWLock)
	}
	l.Mutex.Lock()
}

func (l *Mutex) Unlock() {
	if opts.CheckDeadLock {
		addLockOp(unsafe.Pointer(l), lockOpWUnLock)
	}
	l.Mutex.Unlock()
}

type RWMutex struct {
	sync.RWMutex
}

func (l *RWMutex) RLock() {
	if opts.CheckDeadLock {
		addLockOp(unsafe.Pointer(l), lockOpRLock)
	}
	l.RWMutex.RLock()
}

func (l *RWMutex) RUnlock() {
	if opts.CheckDeadLock {
		addLockOp(unsafe.Pointer(l), lockOpRUnLock)
	}
	l.RWMutex.RUnlock()
}

func (l *RWMutex) Lock() {
	if opts.CheckDeadLock {
		addLockOp(unsafe.Pointer(l), lockOpWLock)
	}
	l.RWMutex.Lock()
}

func (l *RWMutex) Unlock() {
	if opts.CheckDeadLock {
		addLockOp(unsafe.Pointer(l), lockOpWUnLock)
	}
	l.RWMutex.Unlock()
}

const (
	TypeReLock = iota + 1 // a.lock => a.lock
	TypeCyclic            // a.lock => b.lock => c.lock => a.lock
)

const (
	lockOpWLock = iota + 1
	lockOpWUnLock
	lockOpRLock
	lockOpRUnLock
)

type lockOp struct {
	lockInst   unsafe.Pointer
	lockOpType int
	line       string
	stack      string
}

func (op lockOp) lockOp() bool {
	return op.lockOpType == lockOpWLock
}

func (op lockOp) isRLock() bool {
	return op.lockOpType == lockOpRLock
}

func (op lockOp) isWUnLock() bool {
	return op.lockOpType == lockOpWUnLock
}

func (op lockOp) isRUnLock() bool {
	return op.lockOpType == lockOpRUnLock
}

func (op lockOp) isLock() bool {
	return op.lockOpType == lockOpRLock || op.lockOpType == lockOpWLock
}

func (op lockOp) isUnlock() bool {
	return op.lockOpType == lockOpRUnLock || op.lockOpType == lockOpWUnLock
}

func (op lockOp) Equal(tmp lockOp) bool {
	return op.lockInst == tmp.lockInst
}
