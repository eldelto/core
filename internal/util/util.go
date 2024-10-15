package util

import (
	"slices"
	"time"

	"golang.org/x/exp/constraints"
)

func ClampI[A constraints.Integer](value, min, max A) A {
	if value < min {
		return min
	} else if value > max {
		return max
	} else {
		return value
	}
}

func AbsI[A constraints.Integer](value A) A {
	if value < 0 {
		return -value
	}

	return value
}

func ReverseCopy[T any](slice []T) []T {
	result := make([]T, len(slice))
	copy(result, slice)

	slices.Reverse(result)
	return result
}

func SetLocation(t time.Time, loc *time.Location) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), loc)
}
