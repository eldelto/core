package auth

import (
	"context"
	"fmt"

	"github.com/eldelto/core/storage"
)

func userResource(id storage.UserID, parts ...string) Resource {
	parts = append([]string{"/user", id.String()}, parts...)

	r, err := newResource(parts...)
	if err != nil {
		panic(err)
	}

	return r
}

func ListUsers(ctx context.Context, tx storage.ReadTx) ([]*User, error) {
	cap, err := FromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("list users: err=%w", err)
	}

	if err := cap.RequireRead("/user"); err != nil {
		return nil, err
	}

	users, err := storage.ListAll[*User](tx, userBucket)
	if err != nil {
		return nil, fmt.Errorf("list users: err=%w", err)
	}

	return users, nil
}

func GetUser(ctx context.Context, tx storage.ReadTx, id storage.UserID) (*User, error) {
	cap, err := FromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("get user: err=%w", err)
	}

	if err := cap.RequireRead(userResource(id)); err != nil {
		return nil, err
	}

	user, err := storage.Load[*User](tx, userBucket, id[:])
	if err != nil {
		return nil, fmt.Errorf("get user: err=%w", err)
	}

	return user, nil
}
