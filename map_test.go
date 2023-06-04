package hashmaps_test

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/EinfachAndy/hashmaps"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func checkeq[K comparable, V comparable](cm *hashmaps.IHashMap[K, V], get func(k K) (V, bool), t *testing.T) {
	cm.Each(func(key K, val V) bool {
		if ov, ok := get(key); !ok {
			t.Fatalf("key %v should exist", key)
		} else if val != ov {
			t.Fatalf("value mismatch: %v != %v", val, ov)
		}
		v, found := cm.Get(key)
		if !found {
			t.Fatalf("double check failed for key %v", key)
		}
		if v != val {
			t.Fatalf("double check failed for value %v", v)
		}
		return false
	})
}

func TestCrossCheck(t *testing.T) {
	robin := hashmaps.NewRobinHood[uint64, uint32]()

	maps := []hashmaps.IHashMap[uint64, uint32]{
		{
			Get:    robin.Get,
			Put:    robin.Put,
			Remove: robin.Remove,
			Size:   robin.Size,
			Each:   robin.Each,
			Load:   robin.Load,
		},
	}

	const nops = 10000

	for _, m := range maps {
		stdm := make(map[uint64]uint32)
		for i := 0; i < nops; i++ {
			key := uint64(rand.Intn(1000))
			val := rand.Uint32()
			op := rand.Intn(4)

			switch op {
			case 0:
				v1, ok1 := m.Get(key)
				v2, ok2 := stdm[key]
				if ok1 != ok2 || v1 != v2 {
					t.Fatalf("lookup failed")
				}
			case 1:
				// prioritize insert operation
				fallthrough
			case 2:
				_, wasIn := stdm[key]
				stdm[key] = val
				isNew := m.Put(key, val)
				if isNew == wasIn {
					t.Fatalf("Put returned wrong state")
				}

				v, found := m.Get(key)
				if !found {
					t.Fatalf("lookup failed after insert for key %d", key)
				}
				if v != val {
					t.Fatalf("values are not equal %d != %d", v, val)
				}
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
				if !found {
					t.Fatalf("lookup failed for key %d", key)
				}
				wasIn := m.Remove(del)
				if !wasIn {
					t.Fatalf("only deleted keys which are in")
				}
				_, found = m.Get(del)
				if found {
					t.Fatalf("key %d was not removed", key)
				}
			}

			if len(stdm) != m.Size() {
				t.Fatalf("len of maps are not equal %d != %d", len(stdm), m.Size())
			}

			checkeq(&m, func(k uint64) (uint32, bool) {
				v, ok := stdm[k]
				return v, ok
			}, t)
		}
	}
}

func TestCopy(t *testing.T) {
	orig := hashmaps.NewRobinHood[uint64, uint32]()

	for i := uint32(0); i < 10; i++ {
		orig.Put(uint64(i), i)
	}

	cpy := orig.Copy()

	c := hashmaps.IHashMap[uint64, uint32]{Get: cpy.Get, Each: cpy.Each}
	checkeq(&c, orig.Get, t)

	cpy.Put(0, 42)

	if v, _ := cpy.Get(0); v != 42 {
		t.Fatal("didn't get 42")
	}

	if v, _ := orig.Get(0); v != 0 {
		t.Fatal("manipulated origin")
	}
}

func Example() {
	m := hashmaps.NewRobinHood[string, int]()
	m.Put("foo", 42)
	m.Put("bar", 13)

	fmt.Println(m.Get("foo"))
	fmt.Println(m.Get("baz"))

	m.Remove("foo")

	fmt.Println(m.Get("foo"))
	fmt.Println(m.Get("bar"))

	m.Clear()

	fmt.Println(m.Get("foo"))
	fmt.Println(m.Get("bar"))
	// Output:
	// 42 true
	// 0 false
	// 0 false
	// 13 true
	// 0 false
	// 0 false
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
	m := hashmaps.NewRobinHoodWithHasher[dummy, uint32](hasher)
	m.Put(dummy{a: 0, b: 0, c: "", d: 0, e: 0}, 0)
	if m.Size() != 1 {
		t.Fatal("could not insert elem")
	}
}
