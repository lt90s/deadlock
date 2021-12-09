package deadlock

import (
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"unsafe"
)

type lockData struct {
	activeLockOps []lockOp
	graph         *lockGraph
}

var (
	lockDataMutex sync.RWMutex
	lockDataMap   = map[int]*lockData{}
)

func getGid() int {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	gidStr := strings.Split(string(buf[:n]), " ")[1]
	gid, _ := strconv.Atoi(gidStr)
	return gid
}

func getLocalLockData() *lockData {
	gid := getGid()
	lockDataMutex.RLock()
	if data, ok := lockDataMap[gid]; ok {
		lockDataMutex.RUnlock()
		return data
	}
	lockDataMutex.RUnlock()

	lockDataMutex.Lock()
	defer lockDataMutex.Unlock()
	if _, ok := lockDataMap[gid]; !ok {
		lockDataMap[gid] = &lockData{
			graph: newLockGraph(),
		}
	}

	return lockDataMap[gid]
}

func freeLocalLockData() {
	gid := getGid()
	lockDataMutex.Lock()
	defer lockDataMutex.Unlock()
	delete(lockDataMap, gid)
}

func addLockOp(lockInst unsafe.Pointer, lockType int) {
	stack := getStackString()
	_, file, line, _ := runtime.Caller(2)
	lockOp := lockOp{
		lockInst:   lockInst,
		lockOpType: lockType,
		line:       file + ":" + strconv.Itoa(line),
		stack:      stack,
	}

	data := getLocalLockData()

	if lockOp.isLock() {
		for _, activeLockOp := range data.activeLockOps {
			if activeLockOp.Equal(lockOp) {
				opts.OnDeadlock(TypeReLock, []DeadlockInfo{
					{
						lockPath: genLockPath(append(data.activeLockOps, lockOp)),
						stack:    lockOp.stack,
					},
				})
				return
			}
		}
		data.activeLockOps = append(data.activeLockOps, lockOp)
	} else if lockOp.isUnlock() {
		for i := len(data.activeLockOps) - 1; i >= 0; i-- {
			if data.activeLockOps[i].Equal(lockOp) {
				data.activeLockOps = append(data.activeLockOps[:i], data.activeLockOps[i+1:]...)
				break
			}
		}
	}

	if len(data.activeLockOps) == 0 {
		globalGraph.merge(data.graph)
		freeLocalLockData()
		return
	}

	if lockOp.isUnlock() {
		return
	}

	if len(data.activeLockOps) == 1 {
		data.graph.addStartVertices(data.activeLockOps[0].lockInst)
		return
	}

	i := len(data.activeLockOps) - 1
	data.graph.addEdge(data.activeLockOps[i-1], data.activeLockOps[i])
}

func getStackString() string {
	stack := string(debug.Stack())
	lines := strings.Split(stack, "\n")
	for i, line := range lines {
		if strings.Contains(line, "deadlock.addLockOp") {
			lines = lines[i+2:]
		}
	}

	stack = strings.Join(lines, "\n")
	return stack
}

func genLockPath(activeLockOps []lockOp) []string {
	ss := make([]string, 0, len(activeLockOps))
	for _, lockOp := range activeLockOps {
		ss = append(ss, lockOp.line)
	}

	return ss
}
