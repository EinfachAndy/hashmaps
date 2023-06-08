package hashmaps

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"unsafe"
)

// HashFn is a function that returns the hash of 't'.
type HashFn[T any] func(t T) uintptr

// GetHasher returns a hasher for the golang default types.
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
			panic("unsupported integer byte size")
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
		panic(fmt.Sprintf("unsupported key type %T of kind %v", key, kind))
	}
}

var hashByte = func(in uint8) uintptr {
	key := uint32(in)
	key *= 0xcc9e2d51
	key = (key << 15) | (key >> 17)
	key *= 0x1b873593
	return uintptr(key)
}

var hashWord = func(in uint16) uintptr {
	key := uint32(in)
	key *= 0xcc9e2d51
	key = (key << 15) | (key >> 17)
	key *= 0x1b873593
	return uintptr(key)
}

var hashDword = func(key uint32) uintptr {
	key *= 0xcc9e2d51
	key = (key << 15) | (key >> 17)
	key *= 0x1b873593
	return uintptr(key)
}

var hashFloat32 = func(in float32) uintptr {
	p := unsafe.Pointer(&in)
	key := *(*uint32)(p)

	key *= 0xcc9e2d51
	key = (key << 15) | (key >> 17)
	key *= 0x1b873593
	return uintptr(key)
}

var hashFloat64 = func(in float64) uintptr {
	p := unsafe.Pointer(&in)
	key := *(*uint64)(p)

	key ^= (key >> 33)
	key *= 0xff51afd7ed558ccd
	key ^= (key >> 33)
	key *= 0xc4ceb9fe1a85ec53
	key ^= (key >> 33)
	return uintptr(key)
}

// hashQword implements MurmurHash3's 64-bit Finalizer
var hashQword = func(key uint64) uintptr {
	key ^= (key >> 33)
	key *= 0xff51afd7ed558ccd
	key ^= (key >> 33)
	key *= 0xc4ceb9fe1a85ec53
	key ^= (key >> 33)
	return uintptr(key)
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
