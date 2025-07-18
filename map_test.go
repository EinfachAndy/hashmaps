package hashmaps_test

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/EinfachAndy/hashmaps"
	"github.com/EinfachAndy/hashmaps/flat"
	"github.com/EinfachAndy/hashmaps/hopscotch"
	"github.com/EinfachAndy/hashmaps/robin"
	"github.com/EinfachAndy/hashmaps/shared"
	"github.com/EinfachAndy/hashmaps/unordered"
)

var nLoops = 10000

func randString(n int) string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

	b := make([]byte, n)

	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}

	return string(b)
}

func setupMaps[K comparable, V comparable]() []hashmaps.HashMap[K, V] {
	return []hashmaps.HashMap[K, V]{
		*hashmaps.MustNewHashMap(hashmaps.Config[K, V]{
			Type:    hashmaps.Hopscotch,
			MaxLoad: 0.95,
		}),
		*hashmaps.MustNewHashMap(hashmaps.Config[K, V]{
			Type:    hashmaps.Flat,
			MaxLoad: 0.5,
		}),
		*hashmaps.MustNewHashMap(hashmaps.Config[K, V]{
			Type: hashmaps.Unordered,
		}),
		*hashmaps.MustNewHashMap(hashmaps.Config[K, V]{
			Type:    hashmaps.Robin,
			MaxLoad: 0.90,
		}),
	}
}

func checkeq[K comparable, V comparable](
	t *testing.T,
	cm *hashmaps.HashMap[K, V],
	get func(k K) (V, bool),
) {
	cm.Each(func(key K, val V) bool {
		ov, ok := get(key)
		assert.True(t, ok, "key %v should exist", key)
		assert.Equal(t, val, ov, "value mismatch: %v != %v", val, ov)

		v, found := cm.Get(key)
		assert.True(t, found, "double check failed for key %v", key)
		assert.Equal(t, v, val, "double check failed for value %v", v)
		return false
	})
}

func fuzzLoop[K comparable](t *testing.T, n int, getRandKey func(int) K) {
	maps := setupMaps[K, K]()

	for _, m := range maps {
		stdm := make(map[K]K)

		for i := 0; i < n; i++ {
			var (
				key = getRandKey(n / 10)
				val = key
				op  = rand.Intn(4)
			)

			switch op {
			case 0:
				var (
					v1, ok1 = m.Get(key)
					v2, ok2 = stdm[key]
				)

				assert.Equal(t, ok1, ok2, "lookup wrong state")
				assert.Equal(t, v1, v2, "lookup values are different")
			case 1:
				// prioritize insert operation
				fallthrough
			case 2:
				_, wasIn := stdm[key]
				stdm[key] = val
				isNew := m.Put(key, val)
				assert.NotEqual(t, isNew, wasIn, "Put returned wrong state")

				v, found := m.Get(key)
				assert.True(t, found, "lookup failed after insert for key %d", key)
				assert.Equal(t, val, v, "values are not equal %d != %d", v, val)
			case 3:
				var del K

				if len(stdm) == 0 {
					break
				}

				for k := range stdm {
					del = k
					break
				}

				delete(stdm, del)

				_, found := m.Get(del)
				assert.True(t, found, "lookup failed for key %d", key)
				assert.True(t, m.Remove(del))

				_, found = m.Get(del)
				assert.False(t, found, "key %d was not removed", key)
			}

			assert.Equal(t, len(stdm), m.Size(), "len of hashmaps.are not equal %d != %d", len(stdm), m.Size())

			checkeq(t, &m, func(k K) (K, bool) {
				v, ok := stdm[k]
				return v, ok
			})
		}
		t.Log("size:", m.Size(), "Load", m.Load())
	}
}

func TestFuzzInt(t *testing.T) {
	t.Parallel()

	randInt := func(nLoops int) uint64 {
		return uint64(rand.Intn(nLoops/10)) + 1
	}

	fuzzLoop(t, nLoops, randInt)
}

func TestFuzzString(t *testing.T) {
	t.Parallel()

	fuzzLoop(t, nLoops/4, randString)
}

func TestCopy(t *testing.T) {
	orig := robin.New[uint64, uint32]()

	for i := uint32(1); i <= 10; i++ {
		orig.Put(uint64(i), i)
	}

	cpy := orig.Copy()

	c := hashmaps.HashMap[uint64, uint32]{Get: cpy.Get, Each: cpy.Each}
	checkeq(t, &c, orig.Get)

	cpy.Put(0, 42)

	v1, ok1 := cpy.Get(0)
	assert.True(t, ok1)
	assert.Equal(t, uint32(42), v1)

	_, ok2 := orig.Get(0)
	assert.False(t, ok2)
}

func TestCopyFlat(t *testing.T) {
	orig := flat.NewWithHasher[uint64, uint32](11, shared.GetHasher[uint64]())

	for i := uint32(1); i <= 10; i++ {
		orig.Put(uint64(i), i)
	}

	cpy := orig.Copy()

	c := hashmaps.HashMap[uint64, uint32]{Get: cpy.Get, Each: cpy.Each}
	checkeq(t, &c, orig.Get)

	cpy.Put(0, 42)

	v1, ok1 := cpy.Get(0)
	assert.True(t, ok1)
	assert.Equal(t, uint32(42), v1)

	_, ok2 := orig.Get(0)
	assert.False(t, ok2)
}

func TestCopyHopscotch(t *testing.T) {
	orig := hopscotch.New[uint64, uint32]()

	for i := uint32(1); i <= 10; i++ {
		orig.Put(uint64(i), i)
	}

	cpy := orig.Copy()

	c := hashmaps.HashMap[uint64, uint32]{Get: cpy.Get, Each: cpy.Each}
	checkeq(t, &c, orig.Get)

	cpy.Put(0, 42)

	v1, ok1 := cpy.Get(0)
	assert.True(t, ok1)
	assert.Equal(t, uint32(42), v1)

	_, ok2 := orig.Get(0)
	assert.False(t, ok2)
}

func TestCopyUnordered(t *testing.T) {
	orig := unordered.New[uint64, uint32]()

	for i := uint32(1); i <= 10; i++ {
		orig.Put(uint64(i), i)
	}

	cpy := orig.Copy()

	c := hashmaps.HashMap[uint64, uint32]{Get: cpy.Get, Each: cpy.Each}
	checkeq(t, &c, orig.Get)

	cpy.Put(0, 42)

	v1, ok1 := cpy.Get(0)
	assert.True(t, ok1)
	assert.Equal(t, uint32(42), v1)

	_, ok2 := orig.Get(0)
	assert.False(t, ok2)
}

func TestSizes(t *testing.T) {
	t.Parallel()

	maps := setupMaps[int, int]()

	for _, m := range maps {
		for i := 1; i <= 300; i++ {
			m.Put(i, i)

			if m.Size() != i {
				t.Fatal("size invalid")
			}
		}
	}
}

func TestClear(t *testing.T) {
	t.Parallel()

	maps := setupMaps[int, int]()

	for _, m := range maps {
		const nops = 5
		for i := 1; i <= nops; i++ {
			assert.True(t, m.Put(i, i))
		}

		m.Clear()
		assert.Equal(t, 0, m.Size())

		for i := 1; i <= nops; i++ {
			assert.True(t, m.Put(i, i))
		}
	}
}

func TestSimpleUsage(t *testing.T) {
	t.Parallel()

	maps := setupMaps[int, int]()

	for _, m := range maps {
		m.Reserve(7)
		// insert
		assert.Equal(t, 0, m.Size())
		assert.True(t, m.Put(5, 5))
		assert.Equal(t, 1, m.Size())
		assert.False(t, m.Put(5, 5))

		// lookup 5
		v, found := m.Get(5)
		assert.True(t, found)
		assert.Equal(t, 5, v)

		// remove
		assert.True(t, m.Remove(5))
		assert.Equal(t, 0, m.Size())
		assert.False(t, m.Remove(5))

		_, found = m.Get(5)
		assert.False(t, found)
	}
}

func TestUnorderedLookup(t *testing.T) {
	type dummyValue struct {
		test     int
		bigArray [30]int
	}

	var (
		m = unordered.New[string, dummyValue]()
		v = dummyValue{test: 5}
	)

	for i := 0; i < 30; i++ {
		v.bigArray[i] = i
	}
	assert.True(t, m.Put("test", v))

	ptr := m.Lookup("test")
	assert.NotNil(t, ptr)
	assert.Equal(t, v, *ptr)
}

func TestMaxLoad(t *testing.T) {
	t.Parallel()

	maps := setupMaps[int, int]()

	for _, m := range maps {
		assert.Error(t, m.MaxLoad(0.0))
		assert.Error(t, m.MaxLoad(-1.0))
	}
}

func TestComplexKeyType(t *testing.T) {
	t.Parallel()

	type dummy struct {
		a int8
		b uint32
		c string
		d uint64
		e int
	}

	var (
		hasher = func(d dummy) uintptr {
			return 0
		}
		maps = []hashmaps.HashMap[dummy, string]{
			*hashmaps.MustNewHashMap(hashmaps.Config[dummy, string]{
				Type:   hashmaps.Flat,
				Hasher: hasher,
				Empty:  dummy{},
			}),
			*hashmaps.MustNewHashMap(hashmaps.Config[dummy, string]{
				Type:   hashmaps.Robin,
				Hasher: hasher,
			}),
			*hashmaps.MustNewHashMap(hashmaps.Config[dummy, string]{
				Type:   hashmaps.Hopscotch,
				Hasher: hasher,
			}),
			*hashmaps.MustNewHashMap(hashmaps.Config[dummy, string]{
				Type:   hashmaps.Unordered,
				Hasher: hasher,
			}),
		}
	)

	for _, m := range maps {
		isNew := m.Put(dummy{a: 0, b: 0, c: "test", d: 0, e: 0}, "xxx")
		assert.True(t, isNew)
		assert.Equal(t, 1, m.Size())

		val, found := m.Get(dummy{a: 0, b: 0, c: "test", d: 0, e: 0})
		assert.True(t, found)
		assert.Equal(t, "xxx", val)

		_, found = m.Get(dummy{a: 0, b: 0, c: "test1", d: 0, e: 0})
		assert.False(t, found)
	}
}

func TestIterator(t *testing.T) {
	t.Parallel()

	maps := setupMaps[int, int]()

	for _, m := range maps {
		const nops = 10
		for i := 1; i <= nops; i++ {
			assert.True(t, m.Put(i, i))
		}

		var (
			count    = 0
			iterator = func(key int, val int) bool {
				assert.Equal(t, key, val)
				count++

				return false
			}
		)

		m.Each(iterator)
		assert.Equal(t, nops, count)
	}
}

func TestIteratorEarlyTermination(t *testing.T) {
	t.Parallel()

	maps := setupMaps[int, int]()

	for _, m := range maps {
		const nops = 10
		for i := 1; i <= nops; i++ {
			assert.True(t, m.Put(i, i))
		}

		var (
			count    = 0
			iterator = func(key int, val int) bool {
				assert.Equal(t, key, val)
				count++
				return count == 3
			}
		)

		m.Each(iterator)
		assert.Equal(t, 3, count)
	}
}
