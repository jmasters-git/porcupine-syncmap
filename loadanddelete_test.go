package main

import (
	"sync"
	"testing"
)

// When LoadAndDelete is called for a key that is not present,
// it will only perform atomic loads operations,
// thereby demonstrating the Store Buffer litmus test.
func TestLoadAndDelete(t *testing.T) {
	iters := 5_000_000

	var m sync.Map

	for i := range iters {
		var (
			x, y   int64
			r1, r2 int64
			wg     sync.WaitGroup
		)

		wg.Add(2)

		go func() {
			x = 1
			_, _ = m.LoadAndDelete("k")
			r1 = y
			wg.Done()
		}()

		go func() {
			y = 1
			_, _ = m.LoadAndDelete("k")
			r2 = x
			wg.Done()
		}()

		wg.Wait()

		if r1 == 0 && r2 == 0 {
			t.Fatalf("Observed r1=0 && r2=0 in iteration %d of %d", i, iters)
		}
	}

	t.Logf("Did not observe r1=0 && r2=0 in %d iterations", iters)
}

// Delete is just an alias for `_, _ = m.LoadAndDelete(key)`
func TestDelete(t *testing.T) {
	iters := 5_000_000

	var m sync.Map

	for i := range iters {
		var (
			x, y   int64
			r1, r2 int64
			wg     sync.WaitGroup
		)

		wg.Add(2)

		go func() {
			x = 1
			m.Delete("k")
			r1 = y
			wg.Done()
		}()

		go func() {
			y = 1
			m.Delete("k")
			r2 = x
			wg.Done()
		}()

		wg.Wait()

		if r1 == 0 && r2 == 0 {
			t.Fatalf("Observed r1=0 && r2=0 in iteration %d of %d", i, iters)
		}
	}

	t.Logf("Did not observe r1=0 && r2=0 in %d iterations", iters)
}

// Demonstrates that if the key is present, at least one Delete will
// act as a write/"release order" and will never see r1=0 && r2=0.
func TestDeleteWithKeyPresent(t *testing.T) {
	iters := 5_000_000

	var m sync.Map

	for i := range iters {
		var (
			x, y   int64
			r1, r2 int64
			wg     sync.WaitGroup
		)
		// Add key to map, at least one Delete will see the key.
		m.Store("k", 888)

		wg.Add(2)

		go func() {
			x = 1
			m.Delete("k")
			r1 = y
			wg.Done()
		}()

		go func() {
			y = 1
			m.Delete("k")
			r2 = x
			wg.Done()
		}()

		wg.Wait()

		if r1 == 0 && r2 == 0 {
			t.Fatalf("Observed r1=0 && r2=0 in iteration %d of %d", i, iters)
		}
	}

	t.Logf("Did not observe r1=0 && r2=0 in %d iterations", iters)
}

// Demonstrates that `m.Store` provides release ordering preventing the reordering.
func TestStore(t *testing.T) {
	iters := 5_000_000

	var m sync.Map

	for i := range iters {
		var (
			x, y   int64
			r1, r2 int64
			wg     sync.WaitGroup
		)

		wg.Add(2)

		go func() {
			x = 1
			m.Store("k1", i) // Note different keys
			r1 = y
			wg.Done()
		}()

		go func() {
			y = 1
			m.Store("k2", i)
			r2 = x
			wg.Done()
		}()

		wg.Wait()

		if r1 == 0 && r2 == 0 {
			t.Fatalf("Observed r1=0 && r2=0 in iteration %d of %d", i, iters)
		}
	}

	t.Logf("Did not observe r1=0 && r2=0 in %d iterations", iters)
}

// Test Store Buffer litmus test using just Load instead.
func TestLoad(t *testing.T) {
	iters := 5_000_000

	// Note: share the same instance between iterations (each iteration will be ordered by the WaitGroup)
	// I found a per-iteration sync.Map instance does not encounter the reordering.
	// I believe this is most likely due to a per-iteration instance causing cache-misses
	// for every single LoadAndDelete call. Which makes the reorder much less likely to occur.
	var m sync.Map

	for i := range iters {
		var (
			x, y   int64
			r1, r2 int64
			wg     sync.WaitGroup
		)

		wg.Add(2)

		go func() {
			x = 1
			_, _ = m.Load("k")
			r1 = y
			wg.Done()
		}()

		go func() {
			y = 1
			_, _ = m.Load("k")
			r2 = x
			wg.Done()
		}()

		wg.Wait()

		if r1 == 0 && r2 == 0 {
			t.Fatalf("Observed r1=0 && r2=0 in iteration %d of %d", i, iters)
		}
	}

	t.Logf("Did not observe r1=0 && r2=0 in %d iterations", iters)
}

// Same as TestLoad, but using a new sync.Map per iteration to validate the hypothesis
// the creating the sync.Map per-iteration was preventing the reordering to occur
func TestLoadWithPerIterationMap(t *testing.T) {
	iters := 5_000_000

	for i := range iters {
		var (
			m      sync.Map
			x, y   int64
			r1, r2 int64
			wg     sync.WaitGroup
		)

		wg.Add(2)

		go func() {
			x = 1
			_, _ = m.Load("k")
			r1 = y
			wg.Done()
		}()

		go func() {
			y = 1
			_, _ = m.Load("k")
			r2 = x
			wg.Done()
		}()

		wg.Wait()

		if r1 == 0 && r2 == 0 {
			t.Fatalf("Observed r1=0 && r2=0 in iteration %d of %d", i, iters)
		}
	}

	t.Logf("Did not observe r1=0 && r2=0 in %d iterations", iters)
}
