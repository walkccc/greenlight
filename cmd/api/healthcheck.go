package main

import (
	"fmt"
	"net/http"
)

func (app *application) healthcheckHandler(w http.ResponseWriter, r *http.Request) {
	js := `{"status": "available", "environment": %q, "version": %q}`
	js = fmt.Sprintf(js, app.config.env, version)

	w.Header().Set("Content-Type", "application/json")

	_, err := w.Write([]byte(js))
	if err != nil {
		app.logger.Error(err.Error())
		http.Error(w, "The server encountered a problem to write the response.", http.StatusInternalServerError)
		return
	}
}
