package solvent

import "github.com/google/uuid"

// TODO: Implement

type Service struct {
	// db *bbolt.DB
}

func (s *Service) Create() (*Notebook, error) {
	return nil, nil
}

func (s *Service) Fetch(id uuid.UUID) (*Notebook, error) {
	return nil, nil
}

func (s *Service) Update(notebook *Notebook) (*Notebook, error) {
	return nil, nil
}

func (s *Service) Remove(id uuid.UUID) error {
	return nil
}
