package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/justinas/alice"
)

func (app *application) routes() http.Handler {
	router := httprouter.New()

	router.NotFound = http.HandlerFunc(app.notFoundResponse)
	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowedResponse)

	router.HandlerFunc(http.MethodGet, "/v1/healthcheck", app.healthcheckHandler)
	router.HandlerFunc(
		http.MethodGet,
		"/v1/movies",
		app.requirePermission("movies:read", app.getMoviesHandler),
	)
	router.HandlerFunc(
		http.MethodPost,
		"/v1/movies",
		app.requirePermission("movies:write", app.createMovieHandler),
	)
	router.HandlerFunc(
		http.MethodGet,
		"/v1/movies/:id",
		app.requirePermission("movies:read", app.getMovieHandler),
	)
	router.HandlerFunc(
		http.MethodPatch,
		"/v1/movies/:id",
		app.requirePermission("movies:write", app.updateMovieHandler),
	)
	router.HandlerFunc(
		http.MethodDelete,
		"/v1/movies/:id",
		app.requirePermission("movies:write", app.deleteMovieHandler),
	)

	router.HandlerFunc(http.MethodPost, "/v1/users", app.createUserHandler)
	router.HandlerFunc(http.MethodPut, "/v1/users/activated", app.activateUserHandler)

	router.HandlerFunc(
		http.MethodPost,
		"/v1/tokens/authentication",
		app.createAuthenticationTokenHandler,
	)

	standard := alice.New(app.recoverPanic, app.rateLimit, app.authenticate)
	return standard.Then(router)
}
