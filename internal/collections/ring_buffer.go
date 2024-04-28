package collections

type RingBuffer[T any] struct {
	cursor int
	len    int
	buff   []T
}

func NewRingBuffer[T any](len int) RingBuffer[T] {
	return RingBuffer[T]{
		len:  len,
		buff: make([]T, len),
	}
}

func (r *RingBuffer[T]) Append(x T) {
	r.cursor = r.cursor % r.len
	r.buff[r.cursor] = x
	r.cursor++
}

func (r *RingBuffer[T]) Slice() []T {
	slice := make([]T, r.len)
	for i := 0; i < r.len; i++ {
		index := (r.cursor + i) % r.len
		slice[i] = r.buff[index]
	}

	return slice
}
