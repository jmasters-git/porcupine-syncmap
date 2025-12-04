# sync.Map Delete/LoadAndDelete Ordering Issue Demonstration

## Store Buffer Litmus Test

Test file: [loadanddelete_test.go](./loadanddelete_test.go)

This test demonstrates the [Store Buffer litmus test](https://research.swtch.com/hwmm#litmus:~:text=Litmus%20Test%3A%20Write%20Queue%20(also%20called%20Store%20Buffer)) using `sync.Map`. When `LoadAndDelete` (or `Delete`) is called for a key that is not presentin the map, it internally performs only atomic load operations, no writes occur. This means the operation does not provide the release order that an atomic write would, allowing Store Buffer reordering to be observed.

```
Goroutine 1:          Goroutine 2:
x = 1                 y = 1
m.LoadAndDelete("k")  m.LoadAndDelete("k")  // key not present, load-only
r1 = y                r2 = x
```

## Porcupine Test

Test file: [syncmap_test.go](./syncmap_test.go)

Demonstrates a linearizability (porcupine) violation in Go's `sync.Map` under a concurrent `LoadOrStore` + `LoadAndDelete` workload.

- Storing the start/end timestamps of each operation using a plain `time.Since(start).Nanoseconds()` shows violations for non-overlapping operations.
```go
call := time.Since(start).Nanoseconds()

input, output := executeOperation(id, i, &m)

returnTime := time.Since(start).Nanoseconds()
```
- Adding a memory barrier between each timestamp/operation to prevent reordering prevents the issue. See `internal/asm/barrier.go` and `internal/asm/*`.
- Could also replace the ARM64 barrier with a stronger `ISB` instruction barrier to get the same result.
```go
call := time.Since(start).Nanoseconds()
asm.MemoryBarrier() // MFENCE/DMB ISH

input, output := executeOperation(id, i, &m)

asm.MemoryBarrier() // MFENCE/DMB ISH
returnTime := time.Since(start).Nanoseconds()
```
- Other memory ordering operations, such as using `atomic.LoadInt64` and `atomic.StoreInt64` before and after each operation to provide Acquire/Release ordering also prevents the issue.
```go
// Make atomic variable only visible to this individual execution to avoid providing
// any additional synchronization between goroutinues, it should only provide ordering
// for the timestamp memory operations
var atm atomic.Int64
call := time.Since(start).Nanoseconds()
// Go's atomics provide Release-Store ordering on plain/non-atomic operations and Sequential Consistency
// between atomic operations, so the sync.Map memory operations should be ordered after this store.
atm.Store(call)

input, output := executeOperation(id, i, &m)

// Same for the Load(), which orders the timestamp after the map operations with an Acquire-Load ordering.
atm.Load()
returnTime := time.Since(start).Nanoseconds()
```
