package collections

type Set[T comparable] struct {
	m map[T]struct{}
}

func NewSet[T comparable]() Set[T] {
	return Set[T]{m: map[T]struct{}{}}
}

func SetFromSliceValue[T any, O comparable](s []T, f func(t T) O) Set[O] {
	set := NewSet[O]()
	for _, v := range s {
		set.m[f(v)] = struct{}{}
	}

	return set
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

func (s *Set[T]) Contains(key T) bool {
	_, ok := s.m[key]
	return ok
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

func (s *Set[T]) Length() int {
	return len(s.m)
}

func (s *Set[T]) Empty() bool {
	return s.Length() == 0
}

func (s *Set[T]) Remove(key T) {
	delete(s.m, key)
}
