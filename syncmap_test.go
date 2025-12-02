package tests

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/anishathalye/porcupine"
)

func TestSyncMap(t *testing.T) {
	var (
		numRounds = 10000
		numOps    = 50
		workers   = runtime.GOMAXPROCS(0)
	)

	t.Logf("config: rounds=%d ops=%d workers=%d", numRounds, numOps, workers)

	for round := range numRounds {
		var (
			m          sync.Map
			operations []porcupine.Operation
			mu         sync.Mutex
			wg         sync.WaitGroup
			start      = time.Now()
		)

		for g := range workers {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for i := range numOps {
					// var atm atomic.Int64
					call := time.Since(start).Nanoseconds()
					// asm.MemoryBarrier()
					// atm.Store(call)

					input, output := executeOperation(id, i, &m)

					// atm.Load()
					// asm.MemoryBarrier()
					returnTime := time.Since(start).Nanoseconds()

					mu.Lock()
					operations = append(operations, porcupine.Operation{
						ClientId: id,
						Input:    input,
						Call:     call,
						Output:   output,
						Return:   returnTime,
					})
					mu.Unlock()
				}
			}(g)
		}

		wg.Wait()

		result, info := porcupine.CheckOperationsVerbose(Model, operations, 5*time.Second)

		if result == porcupine.Illegal {
			filename := fmt.Sprintf("syncmap_violation_%d_%s.html", round, time.Now().Format("150405"))
			file, err := os.Create(filename)
			if err != nil {
				t.Fatalf("Round %d: failed to create file %s: %v", round, filename, err)
			}
			porcupine.Visualize(Model, info, file)
			file.Close()
			t.Fatalf("Round %d: sync.Map violation saved to %s", round, filename)
		}
	}
	t.Logf("no violation observed after %d rounds", numRounds)
}

func executeOperation(workerID, iter int, m *sync.Map) (SyncMapInput, SyncMapOutput) {
	if iter%3 == 0 { // delete every 3rd op.
		val, ok := m.LoadAndDelete("k")
		if ok {
			return SyncMapInput{op: OpDelete}, SyncMapOutput{found: true, val: val.(int)}
		}
		return SyncMapInput{op: OpDelete}, SyncMapOutput{found: false}
	}

	value := workerID*1000 + iter
	actual, loaded := m.LoadOrStore("k", value)
	if loaded {
		return SyncMapInput{op: OpInsert, val: value}, SyncMapOutput{found: false, val: actual.(int)}
	}
	return SyncMapInput{op: OpInsert, val: value}, SyncMapOutput{found: true}
}

type OpKind int

const (
	OpInsert OpKind = iota
	OpDelete
)

type SyncMapInput struct {
	op  OpKind
	val int
}

type SyncMapOutput struct {
	found bool
	val   int
}

type MapState struct {
	present bool
	val     int
}

var Model = porcupine.Model{
	Init: func() interface{} { return MapState{} },
	Step: func(state, input, output interface{}) (bool, interface{}) {
		st := state.(MapState)
		in := input.(SyncMapInput)
		out := output.(SyncMapOutput)

		switch in.op {
		case OpInsert:
			if st.present {
				if !out.found && out.val == st.val {
					return true, st
				}
				return false, st
			}
			if out.found {
				return true, MapState{present: true, val: in.val}
			}
			return false, st
		case OpDelete:
			if st.present {
				if out.found && out.val == st.val {
					return true, MapState{}
				}
				return false, st
			}
			return !out.found, st
		default:
			return false, st
		}
	},
	DescribeOperation: func(input, output interface{}) string {
		inp := input.(SyncMapInput)
		out := output.(SyncMapOutput)

		switch inp.op {
		case OpInsert:
			if out.found {
				return fmt.Sprintf("Insert(%d) -> ok", inp.val)
			}
			return fmt.Sprintf("Insert(%d) -> key exists (prev %d)", inp.val, out.val)
		case OpDelete:
			if out.found {
				return fmt.Sprintf("Delete() -> deleted (was %d)", out.val)
			}
			return "Delete() -> not found"
		default:
			return "Unknown operation"
		}
	},
}
