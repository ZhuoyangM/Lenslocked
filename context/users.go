package context

import (
	"context"

	"github.com/joncalhoun/lenslocked/models"
)

type key string

const (
	userKey key = "user"
)

func WithUser(ctx context.Context, user *models.User) context.Context {
	return context.WithValue(ctx, userKey, user)
}

func User(ctx context.Context) *models.User {
	tmp := ctx.Value(userKey)
	user, ok := tmp.(*models.User)
	if !ok {
		return nil
	} else {
		return user
	}
}
