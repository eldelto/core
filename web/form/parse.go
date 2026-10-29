package form

import (
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"strconv"
)

var (
	errNotSet = errors.New("required field not set")
)

type ParseError struct {
	Field   string
	Message string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("field %q: %s", e.Field, e.Message)
}

type Parsable interface {
	Parse(r *http.Request)
}

func Parse(r *http.Request, p Parsable) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()
	p.Parse(r)
	return
}

type ParsingFunc[T any] func(r *http.Request, field string) T

func Optional[T any](f ParsingFunc[T]) *T {
	return nil
}

func String(r *http.Request, field string) string {
	s := r.PostFormValue(field)
	if s == "" {
		panic(errNotSet)
	}

	return s
}

func Int(r *http.Request, field string) int {
	raw := String(r, field)
	i, err := strconv.Atoi(raw)
	if err != nil {
		panic(err)
	}

	return i
}

func Float(r *http.Request, field string) float64 {
	raw := String(r, field)
	f, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		panic(err)
	}

	return f
}

func Email(r *http.Request, field string) mail.Address {
	raw := String(r, field)
	email, err := mail.ParseAddress(raw)
	if err != nil {
		panic(err)
	}

	return *email
}
