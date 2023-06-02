package config

import (
	"math/rand"
)

// region ManaBurn ////////////////////////////////////////////////////////////////////////////////////////////////////
func RandomValueArray(seed int64, min, max, length int) []int {
	r := rand.New(rand.NewSource(seed))
	a := r.Perm(length)
	for i, n := range a {
		a[i] = (n % (max - min + 1)) + min
	}
	return a
}

func RandomArrayFromValues(seed int64, values []int, length int) []int {
	a := make([]int, length)
	r := rand.New(rand.NewSource(seed))
	for i := range a {
		a[i] = values[r.Intn(len(values))]
		// a[i] = values[1]
	}
	return a
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
