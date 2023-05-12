package service

import (
	"fmt"

	"github.com/eldelto/core/internal/solvent"
	"github.com/eldelto/core/internal/solvent/service/errcode"
	"github.com/google/uuid"
)

type Repository interface {
	Store(notbook *solvent.Notebook) error
	Update(notebook *solvent.Notebook) error
	Fetch(id uuid.UUID) (*solvent.Notebook, error)
	Remove(id uuid.UUID) error
}

type Service struct {
	repository Repository
}

func NewService(repository Repository) Service {
	return Service{
		repository: repository,
	}
}

// TODO: Wrap returned errors with custom ones
func (s *Service) Create() (*solvent.Notebook, error) {
	notebook, err := solvent.NewNotebook()
	if err != nil {
		return nil, errcode.NewUnknownError(err, "could not create a new notebook")
	}

	err = s.repository.Store(notebook)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	return notebook, nil
}

func (s *Service) Fetch(id uuid.UUID) (*solvent.Notebook, error) {
	return s.repository.Fetch(id)
}

func (s *Service) Update(notebook *solvent.Notebook) (*solvent.Notebook, error) {
	oldNotebook, err := s.Fetch(notebook.ID)
	if err != nil {
		return nil, err
	}

	merged, err := oldNotebook.Merge(notebook)
	if err != nil {
		return nil, errcode.NewNotebookError(notebook.ID, err, "could not merge with old notebook")
	}
	mergedNotebook := merged.(*solvent.Notebook)

	err = s.repository.Update(mergedNotebook)
	if err != nil {
		return nil, err
	}

	return mergedNotebook, nil
}

func (s *Service) Remove(id uuid.UUID) error {
	return s.repository.Remove(id)
}
