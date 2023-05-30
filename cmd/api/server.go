package main

import (
	"fmt"
	"net/http"
	"time"
)

func (app *application) serve() error {
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", app.config.port),
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	app.logger.PrintInfo("starting server", map[string]string{
		"env":  app.config.env,
		"addr": server.Addr,
	})

	return server.ListenAndServe()
}
