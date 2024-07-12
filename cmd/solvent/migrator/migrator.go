package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"os"

	"github.com/eldelto/core/internal/solvent"
	"github.com/google/uuid"
	"go.etcd.io/bbolt"
)

var (
	notebookBucket = "notebooks"
	userID         = uuid.Nil
)

func fetchNotebook(db *bbolt.DB) (*solvent.Notebook, error) {
	var notebook *solvent.Notebook

	err := db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(notebookBucket))
		if bucket == nil {
			return fmt.Errorf("failed to get bucket with name %q", notebookBucket)
		}

		key := userID.String()
		value := bucket.Get([]byte(key))
		if value == nil {
			return fmt.Errorf("original notebook could not be found")
		}

		if err := gob.NewDecoder(bytes.NewBuffer(value)).Decode(&notebook); err != nil {
			return fmt.Errorf("failed to decode todo lists for user %q: %w",
				key, err)
		}

		return nil
	})

	return notebook, err
}

func mapTodoItem(old solvent.ToDoItem) solvent.TodoItem {
	return solvent.TodoItem{
		Checked:   old.Checked,
		CreatedAt: old.OrderValue.UpdatedAt / 1000,
		Title:     old.Title,
	}
}

func mapTodoList(old solvent.ToDoList) (solvent.TodoList, error) {
	new, err := solvent.NewTodoList(old.Title.Value)
	if err != nil {
		return solvent.TodoList{}, err
	}

	new.ID = old.ID
	new.CreatedAt = old.CreatedAt / 1000
	new.UpdatedAt = old.Title.UpdatedAt / 1000

	for _, oldItem := range old.GetItems() {
		newItem := mapTodoItem(oldItem)
		new.Items = append(new.Items, newItem)
	}

	return *new, nil
}

func mapNotebook(old *solvent.Notebook) (*solvent.Notebook2, error) {
	new := solvent.NewNotebook2()
	for _, oldList := range old.GetLists() {
		newList, err := mapTodoList(*oldList)
		if err != nil {
			return nil, err
		}

		new.Lists[oldList.ID] = newList
	}

	return new, nil
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("usage: migrator [bbolt DB path] [output path]")
		os.Exit(-1)
	}

	db, err := bbolt.Open(os.Args[1], 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	newDB, err := bbolt.Open(os.Args[2], 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer newDB.Close()

	originalNotebook, err := fetchNotebook(db)
	if err != nil {
		log.Fatal(err)
	}

	newNotebook, err := mapNotebook(originalNotebook)
	if err != nil {
		log.Fatal(err)
	}

	service, err := solvent.NewService(newDB)
	if err != nil {
		log.Fatal(err)
	}

	_, err = service.UpdateNotebook(uuid.UUID{},
		func(n *solvent.Notebook2) error {
			*n = *newNotebook
			return nil
		})
	if err != nil {
		log.Fatal(err)
	}
}
