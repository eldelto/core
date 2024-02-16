package collections

type Set[T comparable] struct {
	m map[T]struct{}
}

func NewSet[T comparable]() Set[T] {
	return Set[T]{m: map[T]struct{}{}}
}

func SetFromSlice[T comparable](s []T) Set[T] {
	set := NewSet[T]()
	for _, v := range s {
		set.m[v] = struct{}{}
	}

	return set
}

func (s *Set[T]) Slice() []T {
	result := make([]T, len(s.m))
	i := 0
	for k := range s.m {
		result[i] = k
		i++
	}

	return result
}

func (s *Set[T]) Difference(other Set[T]) []T {
	result := []T{}
	for k := range s.m {
		if _, ok := other.m[k]; !ok {
			result = append(result, k)
		}
	}

	return result
}

func (s *Set[T]) Union(other Set[T]) Set[T] {
	result := NewSet[T]()
	for k := range s.m {
		result.m[k] = struct{}{}
	}
	for k := range other.m {
		result.m[k] = struct{}{}
	}

	return result
}
