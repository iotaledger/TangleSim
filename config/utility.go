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
	// r := rand.New(rand.NewSource(seed))
	for i := range a {
		// 24-99: 32.47%
		// 53-99: 15.07%
		// 66-99: 9.98%
		// 82-99: 4.82%
		// value = 0: spammer (i.e., RateSetter() always return true);
		// value = 1: normal;
		// a[i] = values[r.Intn(len(values))]

		// begin := 24
		// a[i] = 1
		// if (i >= begin) {
		// 	a[i] = 0
		// }
		// end := 9
		// if (i > end) {
		// 	a[i] = 0
		// }
		// a[i] = values[r.Intn(len(values))]
		a[i] = 1 // normal
	}
	return a
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
