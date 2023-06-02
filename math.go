package hashmaps

// Ordered is a constraint that permits any ordered type: any type
// that supports the operators < <= >= >.
type Ordered interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64 |
		~string
}

var (
	tab64 = []int{
		63, 0, 58, 1, 59, 47, 53, 2,
		60, 39, 48, 27, 54, 33, 42, 3,
		61, 51, 37, 40, 49, 18, 28, 20,
		55, 30, 34, 11, 43, 14, 22, 4,
		62, 57, 46, 52, 38, 26, 32, 41,
		50, 36, 17, 19, 29, 10, 13, 21,
		56, 45, 25, 31, 35, 16, 9, 12,
		44, 24, 15, 8, 23, 7, 6, 5,
	}
)

// NextPowerOf2 is a fast computation of 2^x
// see: https://stackoverflow.com/questions/466204/rounding-up-to-next-power-of-2
func NextPowerOf2(i uint64) uint64 {
	i--
	i |= i >> 1
	i |= i >> 2
	i |= i >> 4
	i |= i >> 8
	i |= i >> 16
	i |= i >> 32
	i++
	return i
}

// Log2 is a fast computation of log2(x)
// https://stackoverflow.com/questions/11376288/fast-computing-of-log2-for-64-bit-integers
func Log2(value uint64) uint64 {
	value |= value >> 1
	value |= value >> 2
	value |= value >> 4
	value |= value >> 8
	value |= value >> 16
	value |= value >> 32

	index := ((value - (value >> 1)) * 0x07EDD5E59A4E28C2) >> 58
	return uint64(tab64[index])
}

// Max returns the max of a and b.
func Max[T Ordered](a, b T) T {
	if a > b {
		return a
	}
	return b
}

// Min returns the min of a and b.
func Min[T Ordered](a, b T) T {
	if a < b {
		return a
	}
	return b
}
