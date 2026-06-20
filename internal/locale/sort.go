package locale

func compareLess[T any](a, b T, less func(T, T) bool) int {
	if less(a, b) {
		return -1
	}
	if less(b, a) {
		return 1
	}
	return 0
}
