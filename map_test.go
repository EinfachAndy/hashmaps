package hashmaps_test

import (
	"math/rand"
	"testing"
	"time"

	"github.com/EinfachAndy/hashmaps"
	"github.com/stretchr/testify/assert"
)

var nLoops = 1000

func init() {
	rand.Seed(time.Now().UnixNano())
}

func randString(n int) string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

	b := make([]byte, n)

	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}

	return string(b)
}

func setupMaps[K comparable, V comparable]() []hashmaps.IHashMap[K, V] {
	var (
		robin     = hashmaps.NewRobinHood[K, V]()
		unordered = hashmaps.NewUnordered[K, V]()
		flat      = hashmaps.NewFlat[K, V]()
		hopscotch = hashmaps.NewHopscotch[K, V]()
	)

	if err := robin.MaxLoad(0.9); err != nil {
		panic(err.Error())
	}

	if err := hopscotch.MaxLoad(0.95); err != nil {
		panic(err.Error())
	}

	return []hashmaps.IHashMap[K, V]{
		{
			Get:     hopscotch.Get,
			Put:     hopscotch.Put,
			Remove:  hopscotch.Remove,
			Size:    hopscotch.Size,
			Each:    hopscotch.Each,
			Load:    hopscotch.Load,
			Clear:   hopscotch.Clear,
			Reserve: hopscotch.Reserve,
		},
		{
			Get:     flat.Get,
			Put:     flat.Put,
			Remove:  flat.Remove,
			Size:    flat.Size,
			Each:    flat.Each,
			Load:    flat.Load,
			Clear:   flat.Clear,
			Reserve: flat.Reserve,
		},
		{
			Get:     unordered.Get,
			Put:     unordered.Put,
			Remove:  unordered.Remove,
			Size:    unordered.Size,
			Each:    unordered.Each,
			Load:    unordered.Load,
			Clear:   unordered.Clear,
			Reserve: unordered.Reserve,
		},
		{
			Get:     robin.Get,
			Put:     robin.Put,
			Remove:  robin.Remove,
			Size:    robin.Size,
			Each:    robin.Each,
			Load:    robin.Load,
			Clear:   robin.Clear,
			Reserve: robin.Reserve,
		},
	}
}

func checkeq[K comparable, V comparable](
	t *testing.T,
	cm *hashmaps.IHashMap[K, V],
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

func TestCrossCheckInt(t *testing.T) {
	maps := setupMaps[uint64, uint32]()

	for _, m := range maps {
		stdm := make(map[uint64]uint32)

		for i := 0; i < nLoops; i++ {
			var (
				key = uint64(rand.Intn(nLoops/10)) + 1
				val = rand.Uint32()
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
				assert.Equal(t, v, val, "values are not equal %d != %d", v, val)
			case 3:
				var del uint64

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

			assert.Equal(t, len(stdm), m.Size(), "len of maps are not equal %d != %d", len(stdm), m.Size())

			checkeq(t, &m, func(k uint64) (uint32, bool) {
				v, ok := stdm[k]
				return v, ok
			})
		}
		t.Log("size:", m.Size(), "Load", m.Load())
	}
}

func TestCrossCheckString(t *testing.T) {
	maps := setupMaps[string, string]()

	for _, m := range maps {
		stdm := make(map[string]string)

		for i := 0; i < nLoops; i++ {
			var (
				key = randString(rand.Intn(nLoops/10) + 10)
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
				var del string

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

			assert.Equal(t, len(stdm), m.Size(), "len of maps are not equal %d != %d", len(stdm), m.Size())

			checkeq(t, &m, func(k string) (string, bool) {
				v, ok := stdm[k]
				return v, ok
			})
		}
		t.Log("size:", m.Size(), "Load", m.Load())
	}
}

func TestCopyRobin(t *testing.T) {
	orig := hashmaps.NewRobinHood[uint64, uint32]()

	for i := uint32(1); i <= 10; i++ {
		orig.Put(uint64(i), i)
	}

	cpy := orig.Copy()

	c := hashmaps.IHashMap[uint64, uint32]{Get: cpy.Get, Each: cpy.Each}
	checkeq(t, &c, orig.Get)

	cpy.Put(0, 42)

	v1, ok1 := cpy.Get(0)
	assert.True(t, ok1)
	assert.Equal(t, uint32(42), v1)

	_, ok2 := orig.Get(0)
	assert.False(t, ok2)
}

func TestCopyFlat(t *testing.T) {
	orig := hashmaps.NewFlatWithHasher[uint64, uint32](11, hashmaps.GetHasher[uint64]())

	for i := uint32(1); i <= 10; i++ {
		orig.Put(uint64(i), i)
	}

	cpy := orig.Copy()

	c := hashmaps.IHashMap[uint64, uint32]{Get: cpy.Get, Each: cpy.Each}
	checkeq(t, &c, orig.Get)

	cpy.Put(0, 42)

	v1, ok1 := cpy.Get(0)
	assert.True(t, ok1)
	assert.Equal(t, uint32(42), v1)

	_, ok2 := orig.Get(0)
	assert.False(t, ok2)
}

func TestCopyHopscotch(t *testing.T) {
	orig := hashmaps.NewHopscotch[uint64, uint32]()

	for i := uint32(1); i <= 10; i++ {
		orig.Put(uint64(i), i)
	}

	cpy := orig.Copy()

	c := hashmaps.IHashMap[uint64, uint32]{Get: cpy.Get, Each: cpy.Each}
	checkeq(t, &c, orig.Get)

	cpy.Put(0, 42)

	v1, ok1 := cpy.Get(0)
	assert.True(t, ok1)
	assert.Equal(t, uint32(42), v1)

	_, ok2 := orig.Get(0)
	assert.False(t, ok2)
}

func TestSizes(t *testing.T) {
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
		m = hashmaps.NewUnordered[string, dummyValue]()
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
	m := hashmaps.NewRobinHood[int, int]()

	assert.Error(t, m.MaxLoad(0.0))
	assert.Error(t, m.MaxLoad(-1.0))
	assert.Error(t, m.MaxLoad(1.0))
	assert.Error(t, m.MaxLoad(2.0))
}

func TestComplexKeyType(t *testing.T) {
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
		robin     = hashmaps.NewRobinHoodWithHasher[dummy, string](hasher)
		unordered = hashmaps.NewUnorderedWithHasher[dummy, string](hasher)
		flat      = hashmaps.NewFlatWithHasher[dummy, string](dummy{}, hasher)
		maps      = []hashmaps.IHashMap[dummy, string]{
			{
				Get:    flat.Get,
				Put:    flat.Put,
				Remove: flat.Remove,
				Size:   flat.Size,
				Each:   flat.Each,
				Load:   flat.Load,
			},
			{
				Get:    robin.Get,
				Put:    robin.Put,
				Remove: robin.Remove,
				Size:   robin.Size,
				Each:   robin.Each,
				Load:   robin.Load,
			},
			{
				Get:    unordered.Get,
				Put:    unordered.Put,
				Remove: unordered.Remove,
				Size:   unordered.Size,
				Each:   unordered.Each,
				Load:   unordered.Load,
			},
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
