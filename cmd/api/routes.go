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
		app.requireActivatedUser(app.getMoviesHandler),
	)
	router.HandlerFunc(
		http.MethodPost,
		"/v1/movies",
		app.requireActivatedUser(app.createMovieHandler),
	)
	router.HandlerFunc(
		http.MethodGet,
		"/v1/movies/:id",
		app.requireActivatedUser(app.getMovieHandler),
	)
	router.HandlerFunc(
		http.MethodPatch,
		"/v1/movies/:id",
		app.requireActivatedUser(app.updateMovieHandler),
	)
	router.HandlerFunc(
		http.MethodDelete,
		"/v1/movies/:id",
		app.requireActivatedUser(app.deleteMovieHandler),
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
