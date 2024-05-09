package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/eldelto/core/cmd/solvent/migrator/dto"
	"github.com/eldelto/core/internal/solvent"
	"github.com/google/uuid"
	"go.etcd.io/bbolt"
)

func main() {
	if len(os.Args) > 3 {
		fmt.Println("usage: migrator [CSV path] [bbolt DB path]")
		os.Exit(-1)
	}

	file, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	db, err := bbolt.Open(os.Args[2], 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	service, err := solvent.NewService(db)
	if err != nil {
		log.Fatal(err)
	}

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1 // Allow variable number of fields

	data, err := reader.ReadAll()
	if err != nil {
		log.Fatal(err)
	}

	for _, row := range data {
		var userID uuid.UUID
		var notebook *solvent.Notebook
		for i, col := range row {
			switch i {
			case 0:
				id, err := uuid.Parse(col)
				if err != nil {
					log.Fatal(err)
				}
				userID = id
			case 1:
				var notebookDto dto.NotebookDto
				if err := json.Unmarshal([]byte(col), &notebookDto); err != nil {
					log.Fatal(err)
				}
				notebook = dto.NotebookFromDto(&notebookDto)
			}
		}

		if _, err := service.Update(userID, notebook); err != nil {
			log.Fatal(err)
		}
		log.Printf("Migrated notebook %q for user %q.\n", notebook.Identifier(), userID)
	}
}
