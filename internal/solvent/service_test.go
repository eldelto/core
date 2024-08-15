package solvent

import (
	"os"
	"strings"
	"testing"
	"time"

	. "github.com/eldelto/core/internal/testutils"
	"github.com/google/uuid"
	"go.etcd.io/bbolt"
)

const dbPath = "solvent-test.db"

var (
	userID          = uuid.Nil
	plus1hTimestamp = time.Now().Add(1 * time.Hour).UnixMicro()
)

func TestApplyListPatch(t *testing.T) {
	db, err := bbolt.Open(dbPath, 0660, nil)
	AssertNoError(t, err, "bboltOpent")
	defer db.Close()

	service, err := NewService(db, "", "", nil)
	AssertNoError(t, err, "NewService")
	defer os.Remove(dbPath)

	tests := []struct {
		name        string
		createPatch string
		updatePatch string
		want        string
	}{
		{
			name:        "renaming",
			createPatch: "list 1",
			updatePatch: "list 2",
			want:        "list 2",
		},
		{
			name:        "adding items",
			createPatch: "list 1",
			updatePatch: "list 2\nitem 1\nitem 2",
			want:        "list 2\n\n- [ ] item 1\n- [ ] item 2",
		},
		{
			name:        "adding checked items",
			createPatch: "list 1",
			updatePatch: "list 2\n[x] item 1\nitem 2",
			want:        "list 2\n\n- [X] item 1\n- [ ] item 2",
		},
		{
			name:        "checking an existing item",
			createPatch: "list 1\nitem 1",
			updatePatch: "list 2\n[x] item 1\nitem 2",
			want:        "list 2\n\n- [X] item 1\n- [ ] item 2",
		},
		{
			name:        "unchecking an existing item",
			createPatch: "list 1\n[x] item 1",
			updatePatch: "list 2\n[ ] item 1\nitem 2",
			want:        "list 2\n\n- [ ] item 1\n- [ ] item 2",
		},
		{
			name:        "removing items",
			createPatch: "list 1\nitem 1\nitem 2",
			updatePatch: "list 2\nitem 2",
			want:        "list 2\n\n- [ ] item 2",
		},
		{
			name:        "removing all items",
			createPatch: "list 1\nitem 1\nitem 2\nitem 3",
			updatePatch: "list 1",
			want:        "list 1",
		},
		{
			name: "removing and adding items",
			createPatch: `list 1

- [ ] another thing
- [X] thing1
- [X] thing2
- [ ] thing3
- [ ] thing4
- [X] thing5
- [X] thing6
- [X] thing7
- [X] thing9
- [X] thing20
- [ ] bla bla
- [ ] bla bla 2 `,
			updatePatch: `list 1

this
and
that`,
			want: `list 1

- [ ] this
- [ ] and
- [ ] that`,
		},
		{
			name:        "moving items",
			createPatch: "list 1\nitem 1\nitem 2",
			updatePatch: "list 2\nitem 2\nitem 1",
			want:        "list 2\n\n- [ ] item 2\n- [ ] item 1",
		},
		{
			name:        "moving items",
			createPatch: "list 1\nitem 1\nitem 2",
			updatePatch: "list 2\nitem 2\nitem 1",
			want:        "list 2\n\n- [ ] item 2\n- [ ] item 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.FetchNotebook(userID)
			AssertNoError(t, err, "create a new notebook")

			var list TodoList
			_, err = service.UpdateNotebook(userID,
				func(n *Notebook2) error {
					l, err := n.NewList("List 1")
					list = *l
					return err
				})
			AssertNoError(t, err, "create a new list")

			err = service.ApplyListPatch(userID, list.ID, tt.createPatch, plus1hTimestamp)
			AssertNoError(t, err, "apply create list patch")

			err = service.ApplyListPatch(userID, list.ID, tt.updatePatch, plus1hTimestamp)
			AssertNoError(t, err, "apply update list patch")

			list, err = service.FetchTodoList(userID, list.ID)
			AssertNoError(t, err, "get list")

			got := strings.TrimSpace(list.String())
			AssertEquals(t, tt.want, got, "final list state")
		})
	}
}

/*func BenchApplyListPatch(b *testing.B) {
	db, err := bbolt.Open(dbPath, 0660, nil)
	AssertNoError(b, err, "bboltOpent")
	defer db.Close()

	service, err := NewService(db)
	AssertNoError(b, err, "NewService")
	defer os.Remove(dbPath)

	notebook, err := service.Fetch(userID)
	AssertNoError(b, err, "NewNotebook")

	for i := 0; i < 100; i++ {
		list, err := notebook.AddList(fmt.Sprintf("list-%d", i))
		AssertNoError(b, err, "AddList")

		for j := 0; j < 100; j++ {
			itemID, err := list.AddItem(fmt.Sprintf("item-%d", i))
			AssertNoError(b, err, "AddItem")

			_, err = list.CheckItem(itemID)
			AssertNoError(b, err, "CheckItem")

			if i%2 == 0 {
				_, err = list.UncheckItem(itemID)
				AssertNoError(b, err, "UncheckItem")
			}
		}
	}

	_, err = service.store(userID, notebook)
	AssertNoError(b, err, "service.store")

	for n := 0; n < b.N; n++ {

	}
}*/
