package persistence

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/eldelto/core/internal/solvent"
	"github.com/eldelto/core/internal/solvent/service/errcode"
	"github.com/eldelto/core/internal/solvent/web/dto"
	"github.com/google/uuid"

	// Import Postgres driver
	_ "github.com/jackc/pgx/v4/stdlib"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(host, port, user, password string) (*PostgresRepository, error) {
	connectURL := fmt.Sprintf("postgres://%s:%s@%s:%s/solvent", user, password, host, port)
	db, err := sql.Open("pgx", connectURL)
	if err != nil {
		return nil, errcode.NewUnknownError(err, "could not connect to DB")
	}

	repo := PostgresRepository{db: db}

	_, err = repo.db.Exec(`CREATE TABLE IF NOT EXISTS notebooks(
			id VARCHAR(36) PRIMARY KEY NOT NULL, 
			data JSONB NOT NULL
		)`)
	if err != nil {
		db.Close()
		return nil, errcode.NewUnknownError(err, "could not initialize DB")
	}

	return &repo, nil
}

func (r *PostgresRepository) Close() {
	r.db.Close()
}

func (r *PostgresRepository) Store(notebook *solvent.Notebook) error {
	id := notebook.ID.String()
	data, err := notebookToJson(notebook)
	if err != nil {
		return err
	}

	_, err = r.db.Exec("INSERT INTO notebooks VALUES($1, $2)", id, data)

	return errcode.NewNotebookError(notebook.ID, err, "could not execute insert")
}

func (r *PostgresRepository) Update(notebook *solvent.Notebook) error {
	id := notebook.ID.String()
	data, err := notebookToJson(notebook)
	if err != nil {
		return err
	}

	result, err := r.db.Exec("UPDATE notebooks SET data = $2 WHERE id = $1", id, data)
	if err != nil {
		return errcode.NewNotebookError(notebook.ID, err, "could not execute update")
	}

	count, err := result.RowsAffected()
	if err != nil {
		return err
	} else if count <= 0 {
		return errcode.NewNotFoundError("notebook", notebook.ID)
	}

	return nil
}

func (r *PostgresRepository) Fetch(id uuid.UUID) (*solvent.Notebook, error) {
	var data []byte
	err := r.db.QueryRow("SELECT data FROM notebooks WHERE id = $1", id.String()).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, errcode.NewNotFoundError("notebook", id)
	} else if err != nil {
		return nil, errcode.NewNotebookError(id, err, "could not execute select")
	}

	var notebookDto dto.NotebookDto
	json.Unmarshal(data, &notebookDto)
	notebook := dto.NotebookFromDto(&notebookDto)

	return notebook, nil
}

func (r *PostgresRepository) Remove(id uuid.UUID) error {
	_, err := r.db.Exec("DELETE FROM notebooks WHERE id = $1", id.String())
	return errcode.NewNotebookError(id, err, "could not execute delete")
}

func notebookToJson(notebook *solvent.Notebook) ([]byte, error) {
	dto := dto.NotebookToDto(notebook)
	data, err := json.Marshal(dto)
	if err != nil {
		return nil, errcode.NewNotebookError(notebook.ID, err, "could not marshal")
	}

	return data, nil
}
