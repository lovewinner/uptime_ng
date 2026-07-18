package handler

import "strconv"

func positiveIntParam(value string, fallback int) int {
	n, err := strconv.Atoi(value)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func uintParam(value string) (uint, bool) {
	n, err := strconv.ParseUint(value, 10, 0)
	if err != nil || n == 0 {
		return 0, false
	}
	return uint(n), true
}
