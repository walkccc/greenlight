package main

import (
	"fmt"
	"net/http"
)

// createMovieHandler handles requests for "POST /v1/movies".
func (app *application) createMovieHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "create a new movie")
}

// getMovieHandler handles requests for "GET /v1/movies/:id".
func (app *application) getMovieHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	fmt.Fprintf(w, "get the details of movie %d\n", id)
}
