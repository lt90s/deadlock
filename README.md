## 死锁检测

对`sync.RWMutex`和`sync.Mutex`的加锁，解锁方法进行封装，记录每个协程的加锁路径图，并添加到全局加锁路径图中，通过遍历这个图寻找是否存在环路来判定是否存在死锁。
只要代码中的所有加锁路径全部覆盖到，不需要程序实际发生死锁即可检测到死锁。

### 设置

```go
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
```

调用`SetOptions(o Options)`，默认检测到死锁后打印死锁路径和堆栈信息
到标准输出，然后直接退出。

### 使用

支持`RWMtex` 、`Mutex`。使用方式和`sync.RWMutex`、`sync.Mutex` 一致
