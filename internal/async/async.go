package async

import (
	"errors"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

func Parallel[I any, O any](values []I, f func(I) ([]O, error)) ([]O, error) {
	result := []O{}
	mutex := sync.Mutex{}

	group := errgroup.Group{}
	group.SetLimit(10)
	for _, v := range values {
		group.Go(func() error {
			res, err := f(v)
			if err != nil {
				return err
			}

			mutex.Lock()
			defer mutex.Unlock()
			result = append(result, res...)

			return nil
		})
	}

	return result, group.Wait()
}

var ErrTimeout = errors.New("the operation timed out")

func WithTimeout[O any](d time.Duration, f func() (O, error)) (O, error) {
	var result O
	resultChan := make(chan O, 1)
	errChan := make(chan error, 1)
	go func() {
		res, err := f()
		if err != nil {
			errChan <- err
			return
		}
		resultChan <- res
	}()

	select {
	case result = <-resultChan:
		return result, nil
	case err := <-errChan:
		return result, err
	case <-time.After(d):
		return result, ErrTimeout
	}
}
