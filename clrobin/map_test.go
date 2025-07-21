package clrobin_test

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/EinfachAndy/hashmaps/clrobin"
)

type environment struct {
	Map    *clrobin.CLRobin[int, int]
	NumCPU int
	N      int
	sync.WaitGroup
	ShouldTerminate atomic.Bool
	Duration        time.Duration
}

func (e *environment) WaitAndFinish() *environment {
	time.Sleep(e.Duration)
	e.ShouldTerminate.Store(true)
	e.Wait()

	return e
}

func (e *environment) AddEntries(amount int) *environment {
	for i := 0; i < amount; i++ {
		e.Map.Store(i, i)
	}

	return e
}

func newEnvironment() *environment {
	e := &environment{
		Map:      clrobin.New[int, int](),
		NumCPU:   max(runtime.NumCPU(), 2),
		N:        10000,
		Duration: 100 * time.Millisecond,
	}
	e.Add(e.NumCPU)

	return e
}

func runRangeLoop(t *testing.T, e *environment, size int) {
	go func() {
		n := 0
		nn := 0
		for ; !e.ShouldTerminate.Load(); n++ {
			e.Map.Range(func(key, value int) bool {
				assert.Equal(t, key, value)
				assert.True(t, key < size)
				assert.True(t, key >= 0)
				nn++
				return false
			})
		}
		assert.Equal(t, n*size, nn)
		e.Done()
	}()
}

func runReadLoops(t *testing.T, e *environment) {
	e.AddEntries(e.N / 2)

	for i := 0; i < e.NumCPU; i++ {
		go func() {
			defer e.Done()
			for !e.ShouldTerminate.Load() {
				for i := 0; i < e.N; i++ {
					v, loaded := e.Map.Load(i)
					if i < e.N/2 {
						assert.Equal(t, i, v)
						assert.True(t, loaded)
					} else {
						assert.Zero(t, v)
						assert.False(t, loaded)
					}

				}
			}
		}()
	}
}

func runInsertLoops(t *testing.T, e *environment) {
	for i := 1; i <= e.NumCPU; i++ {
		go func(idx int) {
			defer e.Done()
			call := 0
			for !e.ShouldTerminate.Load() {
				for i := 1; i <= e.N; i++ {
					ii := i + (idx * e.N)
					value, loaded := e.Map.Swap(ii, ii)
					if call == 0 {
						assert.False(t, loaded)
						assert.Equal(t, ii, value)
					} else {
						assert.True(t, loaded)
						assert.Equal(t, ii, value)
					}
				}
				call++
			}
		}(i)
	}
}

func runDeleteLoops(t *testing.T, e *environment) {
	for i := 0; i < e.NumCPU; i++ {
		go func(idx int) {
			defer e.Done()
			call := 0
			for !e.ShouldTerminate.Load() {
				for i := 0; i < e.N; i++ {
					ii := i + (idx * e.N)
					value, loaded := e.Map.LoadAndDelete(ii)
					if call == 0 {
						assert.True(t, loaded)
						assert.Equal(t, ii, value)
					} else {
						assert.False(t, loaded)
						assert.Zero(t, value)
					}
				}
				call++
			}
		}(i)
	}
}

func TestConcurrentReads(t *testing.T) {
	e := newEnvironment()

	runReadLoops(t, e)

	e.WaitAndFinish()
}

func TestConcurrentInserts(t *testing.T) {
	e := newEnvironment()
	e.Map.Reserve(2 * uintptr(e.N))

	runInsertLoops(t, e)

	e.WaitAndFinish()
}

func XTestConcurrentInsertsReads(t *testing.T) {
	e := newEnvironment()

	for i := 1; i <= e.NumCPU; i++ {
		go func(idx int) {
			defer e.Done()
			for !e.ShouldTerminate.Load() {
				for i := 1; i <= e.N; i++ {
					ii := i + (idx * e.N)
					e.Map.Store(ii, ii)
					v, found := e.Map.Load(ii)
					assert.True(t, found)
					assert.Equal(t, ii, v)
				}
			}
		}(i)
	}

	e.WaitAndFinish()
}

func XTestConcurrentDelete(t *testing.T) {
	e := newEnvironment()
	e.AddEntries(e.NumCPU * e.N)

	runDeleteLoops(t, e)

	e.WaitAndFinish()
}

func XTestConcurrentInsertsReadsDelete(t *testing.T) {
	e := newEnvironment()

	for i := 1; i <= e.NumCPU; i++ {
		go func(idx int) {
			defer e.Done()
			for !e.ShouldTerminate.Load() {
				for i := 1; i <= e.N; i++ {
					ii := i + (idx * e.N)

					// 1. not found
					_, found := e.Map.Load(ii)
					assert.False(t, found)

					// insert
					e.Map.Store(ii, ii)

					// 2. found
					v, found := e.Map.Load(ii)
					assert.True(t, found)
					assert.Equal(t, ii, v)

					// delete
					v, found = e.Map.LoadAndDelete(ii)
					assert.True(t, found)
					assert.Equal(t, ii, v)

					// 3. not found
					_, found = e.Map.Load(ii)
					assert.False(t, found)
				}
			}
		}(i)
	}

	e.WaitAndFinish()
}

func XTestConcurrentInsertsReadsDeleteMixed(t *testing.T) {
	e := newEnvironment()
	e.Duration = time.Second

	for i := 1; i <= e.NumCPU; i++ {
		go func(idx int) {
			defer e.Done()
			for !e.ShouldTerminate.Load() {
				for i := 1; i <= e.N; i++ {
					value, found := e.Map.Load(i)
					if found {
						assert.Equal(t, i, value)
						old, found := e.Map.LoadAndDelete(i)
						if found {
							assert.Equal(t, i, old)
						}
					} else {
						old, loaded := e.Map.LoadOrStore(i, i)
						if loaded {
							assert.Equal(t, i, old)
						}
					}
				}
			}
		}(i)
	}

	e.WaitAndFinish()
}

func XTestConcurrentRanges(t *testing.T) {
	e := newEnvironment()

	size := 1000
	e.AddEntries(size)

	// concurrent ranges
	for i := 0; i < e.NumCPU; i++ {
		runRangeLoop(t, e, size)
	}

	e.WaitAndFinish()
}

func XTestConcurrentReadsRange(t *testing.T) {
	e := newEnvironment()

	runReadLoops(t, e)

	// range
	e.Add(2)
	runRangeLoop(t, e, e.N/2)
	runRangeLoop(t, e, e.N/2)

	e.WaitAndFinish()
}

func XTestConcurrentInsertAndRange(t *testing.T) {
	e := newEnvironment()

	// range
	go func() {
		for !e.ShouldTerminate.Load() {
			e.Map.Range(func(key, value int) bool {
				assert.Equal(t, key, value)
				return false
			})
		}
	}()

	runInsertLoops(t, e)

	e.WaitAndFinish()
}

func XTestConcurrentDeleteRange(t *testing.T) {
	e := newEnvironment()
	e.AddEntries(e.NumCPU * e.N)

	// range
	go func() {
		for !e.ShouldTerminate.Load() {
			e.Map.Range(func(key, value int) bool {
				assert.Equal(t, key, value)
				return false
			})
		}
	}()

	runDeleteLoops(t, e)

	e.WaitAndFinish()
}
