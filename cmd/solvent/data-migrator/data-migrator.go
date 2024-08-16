package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"os"

	"github.com/eldelto/core/internal/solvent"
	"go.etcd.io/bbolt"
)

const notebookBucket = "notebooks"

type userNotebooks struct {
	userID   string
	notebook solvent.Notebook
}

func listNotebooks(db *bbolt.DB) ([]userNotebooks, error) {
	result := []userNotebooks{}
	err := db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(notebookBucket))
		if bucket == nil {
			return fmt.Errorf("failed to get bucket with name %q", notebookBucket)
		}

		err := bucket.ForEach(func(k, v []byte) error {
			var notebook solvent.Notebook
			if err := gob.NewDecoder(bytes.NewBuffer(v)).Decode(&notebook); err != nil {
				return err
			}

			un := userNotebooks{
				userID:   string(k),
				notebook: notebook,
			}
			result = append(result, un)

			return nil
		})

		return err
	})

	return result, err
}

func copyLists(db *bbolt.DB, source, destination string) error {
	return db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(notebookBucket))
		if bucket == nil {
			return fmt.Errorf("failed to get bucket with name %q", notebookBucket)
		}

		value := bucket.Get([]byte(source))
		if value == nil {
			return fmt.Errorf("user %q does not exist", source)
		}
		var sourceNotebook solvent.Notebook
		if err := gob.NewDecoder(bytes.NewBuffer(value)).Decode(&sourceNotebook); err != nil {
			return err
		}

		value = bucket.Get([]byte(destination))
		if value == nil {
			return fmt.Errorf("user %q does not exist", destination)
		}
		var destinationNotebook solvent.Notebook
		if err := gob.NewDecoder(bytes.NewBuffer(value)).Decode(&destinationNotebook); err != nil {
			return err
		}

		for k, v := range sourceNotebook.Lists {
			destinationNotebook.Lists[k] = v
		}

		buffer := bytes.Buffer{}
		if err := gob.NewEncoder(&buffer).Encode(destinationNotebook); err != nil {
			return fmt.Errorf("failed to encode notebook for user %q: %w", destination, err)
		}
		if err := bucket.Put([]byte(destination), buffer.Bytes()); err != nil {
			return fmt.Errorf("failed to persist notebook for user %q: %w", destination, err)
		}

		return nil
	})
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: data-migrator [path]")
		return
	}

	dbPath := os.Args[1]

	db, err := bbolt.Open(dbPath, 0600, nil)
	if err != nil {
		log.Fatalf("failed to open bbolt DB %q: %v", dbPath, err)
	}
	defer db.Close()

	notebooks, err := listNotebooks(db)
	if err != nil {
		log.Fatalf("failed to fetch notebooks: %v", err)
	}

	for _, n := range notebooks {
		fmt.Printf("user: %s with %d lists\n", n.userID, len(n.notebook.Lists))
	}

	fmt.Println("Which lists do you want to migrate?")

	var source, destination string
	fmt.Println("Source user ID:")
	if _, err := fmt.Scanln(&source); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Destination user ID:")
	if _, err := fmt.Scanln(&destination); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Copying lists from user %q to user %q ... \n", source, destination)
	if err := copyLists(db, source, destination); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Done!")
}
