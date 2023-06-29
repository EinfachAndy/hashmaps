package hashmaps

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
