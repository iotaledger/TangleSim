package config

import "math/rand"

// region ManaBurn ////////////////////////////////////////////////////////////////////////////////////////////////////
func RandomValueArray(seed int64, min, max, length int) []int {
	r := rand.New(rand.NewSource(seed))
	a := r.Perm(length)
	for i, n := range a {
		a[i] = (n % (max - min + 1)) + min
	}
	return a
}

func ZeroValueArray(length int) []float64 {
	return make([]float64, length)
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
