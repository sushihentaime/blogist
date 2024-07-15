package main

import (
	"context"
	"net/http"

	"github.com/sushihentaime/blogist/internal/userservice"
)

type contextKey string

const userContextKey = contextKey("user")

func (app *application) createUserContext(r *http.Request, user *userservice.User) *http.Request {
	ctx := context.WithValue(r.Context(), userContextKey, user)
	return r.WithContext(ctx)
}

func (app *application) getUserContext(r *http.Request) *userservice.User {
	user, ok := r.Context().Value(userContextKey).(*userservice.User)
	if !ok {
		return nil
	}
	return user
}
