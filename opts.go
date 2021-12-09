package deadlock

import (
	"fmt"
	"os"
	"strings"
)

type (
	DeadlockInfo struct {
		stack    string
		lockPath []string
	}
	OnDeadlockFunc func(deadlockType int, infos []DeadlockInfo)
	Options        struct {
		CheckDeadLock bool
		OnDeadlock    OnDeadlockFunc
	}
)

var (
	opts = Options{
		CheckDeadLock: false,
		OnDeadlock:    defaultOnDeadlock,
	}
)

func SetOptions(o Options) {
	if !o.CheckDeadLock {
		return
	}

	opts.CheckDeadLock = true

	if o.OnDeadlock != nil {
		opts.OnDeadlock = o.OnDeadlock
	}

	lockDataMutex.Lock()
	lockDataMap = map[int]*lockData{}
	lockDataMutex.Unlock()
}

func defaultOnDeadlock(typ int, infos []DeadlockInfo) {
	fmt.Println(DefaultDeadlockInfoString(typ, infos))
	os.Exit(-1)
}

func DefaultDeadlockInfoString(typ int, infos []DeadlockInfo) string {
	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf("deadlockType:%d\n", typ))
	for i, info := range infos {
		builder.WriteString(fmt.Sprintf("#################goroutine %d info#################\n", i))
		builder.WriteString(fmt.Sprintf("lockPath:\n%s\n", strings.Join(info.lockPath, "\n")))
		builder.WriteString(fmt.Sprintf("stack:\n%s\n", info.stack))
	}
	return builder.String()
}
