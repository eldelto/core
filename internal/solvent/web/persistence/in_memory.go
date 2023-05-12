package persistence

import (
	"sync"

	"github.com/eldelto/core/internal/solvent"
	"github.com/eldelto/core/internal/solvent/service/errcode"
	"github.com/google/uuid"
)

type NotebookStore map[uuid.UUID]solvent.Notebook

type InMemoryRepository struct {
	store NotebookStore
	mutex sync.Mutex
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		store: NotebookStore{},
		mutex: sync.Mutex{},
	}
}

func (r *InMemoryRepository) Store(notebook *solvent.Notebook) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.store[notebook.ID] = *notebook
	return nil
}

func (r *InMemoryRepository) Update(notebook *solvent.Notebook) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	_, ok := r.store[notebook.ID]
	if !ok {
		return errcode.NewNotFoundError("notebook", notebook.ID)
	}
	r.store[notebook.ID] = *notebook

	return nil
}

func (r *InMemoryRepository) Fetch(id uuid.UUID) (*solvent.Notebook, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	notebook, ok := r.store[id]
	if !ok {
		return nil, errcode.NewNotFoundError("notebook", notebook.ID)
	}

	return &notebook, nil
}

func (r *InMemoryRepository) Remove(id uuid.UUID) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	delete(r.store, id)

	return nil
}
