package errcode

import (
	"fmt"

	"github.com/google/uuid"
)

// NotFoundError indicates that a ToDoListItem with the given ID
// does not exist
type NotFoundError struct {
	Kind    string
	ID      uuid.UUID
	message string
}

func NewNotFoundError(kind string, id uuid.UUID) error {
	return &NotFoundError{
		Kind:    kind,
		ID:      id,
		message: fmt.Sprintf("%s with ID '%v' could not be found", kind, id),
	}
}

func (e *NotFoundError) Error() string {
	return e.message
}

type NotebookError struct {
	ID      uuid.UUID
	err     error
	message string
}

func NewNotebookError(id uuid.UUID, err error, message string) error {
	if err == nil {
		return nil
	}

	return &NotebookError{
		ID:      id,
		err:     err,
		message: fmt.Sprintf("%s [notebook ID: '%s']", message, id.String()),
	}
}

func (e *NotebookError) Error() string {
	return e.message
}

func (e *NotebookError) Unwrap() error {
	return e.err
}

// UnknownError indicates an unhandled error from another library that
// gets wrapped
type UnknownError struct {
	err     error
	message string
}

func NewUnknownError(err error, message string) error {
	if err == nil {
		return nil
	}

	return &UnknownError{
		err:     err,
		message: message,
	}
}

func (e *UnknownError) Error() string {
	return e.message
}

func (e *UnknownError) Unwrap() error {
	return e.err
}
