package omniorg

import (
	_ "embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"text/template"
	"time"

	"github.com/eldelto/core/auth"
	"github.com/eldelto/core/storage"
)

type (
	ID             string
	Type           string
	Status         uint
	ExternalID     string
	ExternalStatus uint
)

const (
	StatusNew = Status(iota)
	StatusTakenOver
	StatusIrrelevant

	ExternalStatusOpen = ExternalStatus(iota)
	ExternalStatusClosed
)

var (
	//go:embed org-item.tmpl.org
	orgItem         string
	orgItemTemplate = template.Must(template.New("org-item").Parse(orgItem))

	sources []Source = []Source{}
	OrgDir           = "."
	repo             = storage.Require(filepath.Join(OrgDir, "omni-org.db"))
)

type Action func(i *Item) error

type Item struct {
	ID             ID
	Type           Type
	Status         Status
	ExternalID     ExternalID
	ExternalStatus ExternalStatus
	Title          string
	Content        string
	UpdatedAt      time.Time
	ScheduledAt    *time.Time
	Actions        map[string]Action
}

func (i *Item) BucketKey() []byte {
	return []byte(i.ID)
}

type Source interface {
	List(from, until time.Time) ([]ExternalID, error)
	FetchItem(id ExternalID) (Item, error)
}

func RegisterSource(s Source) {
	sources = append(sources, s)
}

func toOrgItem(i *Item, w io.Writer) error {
	if err := orgItemTemplate.Execute(w, i); err != nil {
		return fmt.Errorf("to org item: item=%#v, err=%w", i, err)
	}
	return nil
}

func writeItems(s Source, w io.Writer, from, until time.Time) error {
	ids, err := s.List(from, until)
	if err != nil {
		return fmt.Errorf("write items: err=%w", err)
	}

	for _, id := range ids {
		item, err := s.FetchItem(id)
		if err != nil {
			return err
		}

		if err := toOrgItem(&item, w); err != nil {
			return err
		}
	}

	return nil
}

func GenerateOrgFile(from, until time.Time) error {
	path := filepath.Join(OrgDir, "omni.org")
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create org file: path=%q, err=%w", path, err)
	}
	defer f.Close()

	if _, err := io.WriteString(f, "* Omni"); err != nil {
		return fmt.Errorf("write org file heading: err=%w", err)
	}

	for _, s := range sources {
		if err := writeItems(s, f, from, until); err != nil {
			return err
		}
	}

	return nil
}

func loadItem(tx *storage.Tx, id ID) (*Item, error) {
	item, err := storage.Load[*Item](tx, "items", []byte(id))
	if err != nil {
		return nil, fmt.Errorf("load item: id=%q, err=%w", err)
	}
	return item, nil
}

func updateItem(tx *storage.Tx, item Item) error {
	if err := storage.Store(tx, "items", item, auth.UserID{}); err != nil {
		return fmt.Errorf("update item: id=%q, err=%w", err)
	}
	return nil
}

func MarkTakenOver(id ID) error {
	repo.Write(func(tx *storage.Tx) error {
		item, err := loadItem(tx, id)
		if err != nil {
			return err
		}

		item.Status = StatusTakenOver
		storage.Store(tx, item, auth.UserID{})

		return nil
	})

}

// func fromOrgItem(org string) (Item, error) {
// 	panic("implement")
// 	return Item{}, nil
// }
