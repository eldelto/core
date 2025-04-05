package errs

import "fmt"

type ErrNotFound struct {
	ID       string
	Resource string
}

func NotFound(id, resource string) error {
	return &ErrNotFound{
		ID:       id,
		Resource: resource,
	}
}

func (e *ErrNotFound) Error() string {
	return fmt.Sprintf("%s with ID %q not found", e.Resource, e.ID)
}

func (e *ErrNotFound) Is(err error) bool {
	_, ok := err.(*ErrNotFound)
	return ok
}

type ErrNotAuthenticated struct {
	Resource string
}

func NotAuthenticated(resource string) error {
	return &ErrNotAuthenticated{
		Resource: resource,
	}
}

func (e *ErrNotAuthenticated) Error() string {
	return fmt.Sprintf("no authentication found while accessing resource %q",
		e.Resource)
}

func (e *ErrNotAuthenticated) Is(err error) bool {
	_, ok := err.(*ErrNotAuthenticated)
	return ok
}
