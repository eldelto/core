package solvent

import "github.com/google/uuid"

// TODO: Implement

type Service struct {
	// db *bbolt.DB
}

func (s *Service) Create() (*Notebook, error) {
	panic("not implemented")
}

func (s *Service) Fetch(id uuid.UUID) (*Notebook, error) {
	panic("not implemented")
}

func (s *Service) Update(notebook *Notebook) (*Notebook, error) {
	panic("not implemented")
}

func (s *Service) Remove(id uuid.UUID) error {
	panic("not implemented")
}
