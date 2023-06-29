package hashmaps_test

import (
	"math/rand"
	"testing"
	"time"

	"github.com/EinfachAndy/hashmaps"
	"github.com/stretchr/testify/assert"
)

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
	robin.MaxLoad(0.9)

	return []hashmaps.IHashMap[K, V]{
		{
			Get:    hopscotch.Get,
			Put:    hopscotch.Put,
			Remove: hopscotch.Remove,
			Size:   hopscotch.Size,
			Each:   hopscotch.Each,
			Load:   hopscotch.Load,
			Clear:  hopscotch.Clear,
		},
		{
			Get:    flat.Get,
			Put:    flat.Put,
			Remove: flat.Remove,
			Size:   flat.Size,
			Each:   flat.Each,
			Load:   flat.Load,
			Clear:  flat.Clear,
		},
		{
			Get:    unordered.Get,
			Put:    unordered.Put,
			Remove: unordered.Remove,
			Size:   unordered.Size,
			Each:   unordered.Each,
			Load:   unordered.Load,
			Clear:  unordered.Clear,
		},
		{
			Get:    robin.Get,
			Put:    robin.Put,
			Remove: robin.Remove,
			Size:   robin.Size,
			Each:   robin.Each,
			Load:   robin.Load,
			Clear:  robin.Clear,
		},
	}
}

func checkeq[K comparable, V comparable](cm *hashmaps.IHashMap[K, V], get func(k K) (V, bool), t *testing.T) {
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
	const nops = 1000
	for _, m := range maps {
		stdm := make(map[uint64]uint32)
		for i := 0; i < nops; i++ {
			key := uint64(rand.Intn(1000)) + 1
			val := rand.Uint32()
			op := rand.Intn(4)

			switch op {
			case 0:
				v1, ok1 := m.Get(key)
				v2, ok2 := stdm[key]
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

			checkeq(&m, func(k uint64) (uint32, bool) {
				v, ok := stdm[k]
				return v, ok
			}, t)
		}
		t.Log("size:", m.Size(), "Load", m.Load())
	}
}

func TestCrossCheckString(t *testing.T) {

	maps := setupMaps[string, string]()
	const nops = 1000
	for _, m := range maps {
		stdm := make(map[string]string)
		for i := 0; i < nops; i++ {
			key := randString(rand.Intn(40) + 10)
			val := key
			op := rand.Intn(4)

			switch op {
			case 0:
				v1, ok1 := m.Get(key)
				v2, ok2 := stdm[key]
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

			checkeq(&m, func(k string) (string, bool) {
				v, ok := stdm[k]
				return v, ok
			}, t)
		}
		t.Log("size:", m.Size(), "Load", m.Load())
	}
}

func TestCopy(t *testing.T) {
	orig := hashmaps.NewRobinHood[uint64, uint32]()

	for i := uint32(1); i <= 10; i++ {
		orig.Put(uint64(i), i)
	}

	cpy := orig.Copy()

	c := hashmaps.IHashMap[uint64, uint32]{Get: cpy.Get, Each: cpy.Each}
	checkeq(&c, orig.Get, t)

	cpy.Put(0, 42)

	v1, ok1 := cpy.Get(0)
	assert.True(t, ok1)
	assert.Equal(t, uint32(42), v1)

	_, ok2 := orig.Get(0)
	assert.False(t, ok2)
}

func TestSizes(t *testing.T) {
	maps := setupMaps[int, int]()
	const nops = 300
	for _, m := range maps {
		for i := 1; i <= nops; i++ {
			m.Put(i, i)
			if m.Size() != i {
				t.Fatal("size invalid")
			}
		}
	}
}

func TestClear(t *testing.T) {
	maps := setupMaps[int, int]()
	const nops = 5
	for _, m := range maps {
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

func TestComplexKeyType(t *testing.T) {
	type dummy struct {
		a int8
		b uint32
		c string
		d uint64
		e int
	}
	hasher := func(d dummy) uintptr {
		return 0
	}
	robin := hashmaps.NewRobinHoodWithHasher[dummy, string](hasher)
	unordered := hashmaps.NewUnorderedWithHasher[dummy, string](hasher)
	flat := hashmaps.NewFlatWithHasher[dummy, string](dummy{}, hasher)
	maps := []hashmaps.IHashMap[dummy, string]{
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
