package validate

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/eldelto/core/storage"
)

var (
	ErrTypeMismatch = errors.New("cannot cast to the correct type")
)

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("field %q: %s", e.Field, e.Message)
}

type Validatable interface {
	Validate() error
}

func newErr(field, msg string) error {
	return &ValidationError{Field: field, Message: msg}
}

func True(field string, condition bool, msg string) error {
	if !condition {
		return newErr(field, msg)
	}

	return nil
}

func NotEmpty(field string, data any) error {
	if data == nil {
		return newErr(field, "cannot be nil")
	}

	v := reflect.ValueOf(data)
	switch v.Kind() {
	case reflect.String, reflect.Slice, reflect.Map, reflect.Chan:
		if v.Len() < 1 {
			return newErr(field, "cannot be empty")
		}
	}

	return nil
}

func NotEqual(field string, data, other any) error {
	return True(field, data != other, fmt.Sprintf("%v should not equal %v", data, other))
}

func BeforeCommit[T Validatable](data T) storage.PreCommitFunc {
	return func(tx storage.WriteTx, bucket string, old, new any) error {
		data, ok := new.(T)
		if !ok {
			return fmt.Errorf("validation failed: type=%T, err=%w",
				new, ErrTypeMismatch)
		}

		return data.Validate()
	}
}
