package hashmaps

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"unsafe"
)

// HashFn is a function that returns the hash of 't'.
type HashFn[T any] func(t T) uintptr

const (
	prime1 uint64 = 11400714785074694791
	prime2 uint64 = 14029467366897019727
	prime3 uint64 = 1609587929392839161
)

// GetHasher returns a hasher for the given type
func GetHasher[Key any]() HashFn[Key] {
	var key Key
	kind := reflect.ValueOf(&key).Elem().Type().Kind()

	switch kind {
	case reflect.Int, reflect.Uint, reflect.Uintptr:
		switch unsafe.Sizeof(key) {
		case 2:
			return *(*func(Key) uintptr)(unsafe.Pointer(&hashWord))
		case 4:
			return *(*func(Key) uintptr)(unsafe.Pointer(&hashDword))
		case 8:
			return *(*func(Key) uintptr)(unsafe.Pointer(&hashQword))

		default:
			panic(fmt.Errorf("unsupported integer byte size"))
		}

	case reflect.Int8, reflect.Uint8:
		return *(*func(Key) uintptr)(unsafe.Pointer(&hashByte))
	case reflect.Int16, reflect.Uint16:
		return *(*func(Key) uintptr)(unsafe.Pointer(&hashWord))
	case reflect.Int32, reflect.Uint32:
		return *(*func(Key) uintptr)(unsafe.Pointer(&hashDword))
	case reflect.Int64, reflect.Uint64:
		return *(*func(Key) uintptr)(unsafe.Pointer(&hashQword))
	case reflect.Float32:
		return *(*func(Key) uintptr)(unsafe.Pointer(&hashFloat32))
	case reflect.Float64:
		return *(*func(Key) uintptr)(unsafe.Pointer(&hashFloat64))
	case reflect.String:
		return *(*func(Key) uintptr)(unsafe.Pointer(&fnv1aModified))

	default:
		panic(fmt.Errorf("unsupported key type %T of kind %v", key, kind))
	}
}

var hashByte = func(key uint8) uintptr {
	h := uint64(key) * prime1
	h ^= h >> 33
	h *= prime2
	h ^= h >> 29
	return uintptr(h)
}

var hashWord = func(key uint16) uintptr {
	h := uint64(key) * prime1
	h ^= h >> 33
	h *= prime2
	h ^= h >> 29
	return uintptr(h)
}

var hashDword = func(key uint32) uintptr {
	h := uint64(key) * prime1
	h ^= h >> 33
	h *= prime2
	h ^= h >> 29
	return uintptr(h)
}

var hashFloat32 = func(key float32) uintptr {
	p := unsafe.Pointer(&key)
	b := *(*uint32)(p)

	h := uint64(b) * prime1
	h ^= h >> 33
	h *= prime2
	h ^= h >> 29
	return uintptr(h)
}

var hashFloat64 = func(key float64) uintptr {
	p := unsafe.Pointer(&key)
	b := *(*uint64)(p)

	h := b * prime1
	h ^= h >> 33
	h *= prime2
	h ^= h >> 29
	h *= prime3
	h ^= h >> 32
	return uintptr(h)
}

var hashQword = func(key uint64) uintptr {
	h := key * prime1
	h ^= h >> 33
	h *= prime2
	h ^= h >> 29
	h *= prime3
	h ^= h >> 32
	return uintptr(h)
}

// fnv1aModified implements a simpler and faster variant of the fnv1a algorithm, that seems good enough for string hashing.
var fnv1aModified = func(b []byte) uintptr {
	const prime64 = uint64(1099511628211)
	h := uint64(14695981039346656037)

	for len(b) >= 8 {
		x := binary.BigEndian.Uint32(b)
		b = b[4:]
		y := binary.BigEndian.Uint32(b)
		b = b[4:]
		z := (uint64(x) << 32) | uint64(y)
		h = (h ^ z) * prime64
	}

	if len(b) >= 4 {
		x := binary.BigEndian.Uint16(b)
		b = b[2:]
		y := binary.BigEndian.Uint16(b)
		b = b[2:]
		z := (uint64(x) << 16) | uint64(y)
		h = (h ^ z) * prime64
	}

	if len(b) >= 2 {
		h = (h ^ uint64(b[0]^b[1])) * prime64
		b = b[2:]
	}

	if len(b) > 0 {
		h = (h ^ uint64(b[0])) * prime64
	}

	return uintptr(h)
}
