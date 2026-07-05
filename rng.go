package main

import "math/rand"

// uniform returns a random float in [a, b), matching Python's random.uniform.
func uniform(a, b float64) float64 {
	return a + rand.Float64()*(b-a)
}

// choiceStr returns a random element of the given slice.
func choiceStr(options []string) string {
	return options[rand.Intn(len(options))]
}

// randIntn returns a random int in [0, n).
func randIntn(n int) int {
	if n <= 0 {
		return 0
	}
	return rand.Intn(n)
}
