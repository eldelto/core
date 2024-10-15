package cache

import (
	"sync"
)

type Cacher[T any] interface {
	GetOrElse(key string, fallback func() (T, error)) (T, error)
}

type OneTime[T any] struct {
	lock  sync.RWMutex
	cache map[string]T
}

func NewOneTime[T any]() Cacher[T] {
	return &OneTime[T]{
		lock:  sync.RWMutex{},
		cache: map[string]T{},
	}
}

func (ot *OneTime[T]) GetOrElse(key string, fallback func() (T, error)) (T, error) {
	ot.lock.RLock()
	value, ok := ot.cache[key]
	if ok {
		ot.lock.RUnlock()
		return value, nil
	}
	ot.lock.RUnlock()

	ot.lock.Lock()
	defer ot.lock.Unlock()

	value, err := fallback()
	if err != nil {
		return value, err
	}
	ot.cache[key] = value

	return value, nil
}
