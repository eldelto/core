package solvent

import "github.com/google/uuid"

// TODO: Implement

type Service struct {
	// db *bbolt.DB
}

var notebook, _ = NewNotebook()

func (s *Service) Create() (*Notebook, error) {
	panic("not implemented")
}

func (s *Service) Fetch(id uuid.UUID) (*Notebook, error) {

	list, err := notebook.AddList("Mock List")
	if err != nil {
		return nil, err
	}

	list.AddItem("Item 1")
	list.AddItem("Item 2")

	return notebook, nil
}

func (s *Service) Update(notebook *Notebook) (*Notebook, error) {
	panic("not implemented")
}

func (s *Service) Remove(id uuid.UUID) error {
	panic("not implemented")
}
