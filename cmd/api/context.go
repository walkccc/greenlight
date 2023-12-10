package main

import (
	"context"
	"net/http"

	"github.com/walkccc/greenlight/internal/data"
)

type contextKey string

const userContextKey = contextKey("user")

// contextSetUser returns a new copy of the request with the provided User
// struct added to the context. Note that we use our userContextKey constant as
// the key.
func (app *application) contextSetUser(r *http.Request, user *data.User) *http.Request {
	ctx := context.WithValue(r.Context(), userContextKey, user)
	return r.WithContext(ctx)
}

// contextGetUser retrieves the User struct from the request.
func (app *application) contextGetUser(r *http.Request) *data.User {
	user, ok := r.Context().Value(userContextKey).(*data.User)
	if !ok {
		panic("Missing user value in request context.")
	}
	return user
}
