package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// serve starts a server. When we receive a SIGINT or SIGTERM signal, we
// instruct our server to stop accepting any new HTTP requests, and give any
// in-flight requests a 'grace period' of 30 seconds to complete before the
// application is terminated.
func (app *application) serve() error {
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", app.config.port),
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		ErrorLog:     slog.NewLogLogger(app.logger.Handler(), slog.LevelError),
	}

	// shutdownError is a channel that receives any errors returned by the
	// graceful Showtdown().
	shutdownError := make(chan error)

	// Start a background goroutine.
	go func() {
		// Create a quit channel which carries os.Signal values.
		quit := make(chan os.Signal, 1)

		// Use signal.Notify() to listen for incoming SIGINT and SIGTERM signals and
		// relay them to the quit channel. Any other signals will not be caught by
		// signal.Notify() and will retain their default behavior.
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		// Read the signal from the quit channel. This code will block until a
		// signal is received.
		s := <-quit

		app.logger.Info("Shutting down server.", "signal", s.String())

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Call Shutdown() on our server, passing in the context we just made.
		// Shutdown() will return nil if the graceful shutdown was successful, or an
		// error (which may happen because of a problem closing the listeners, or
		// because the shutdown didn't complete before the 30-second context
		// deadline is hit). We relay this return value to the shutdownError
		// channel.
		shutdownError <- server.Shutdown(ctx)
	}()

	app.logger.Info("Starting server.", "addr", server.Addr, "env", app.config.env)

	// Calling Shutdown() on our server will cause ListenAndServe() to immediately
	// return a http.ErrServerClosed error. So if we see this error, it is
	// actually a good thing and an indication that the graceful shutdown has
	// started. So we check specifically for this, only returning the error if it
	// is NOT http.ErrServerClosed.
	err := server.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	// Otherwise, we wait to receive the return value from Shutdown() on the
	// shutdownError channel. If return value is an error, we know that there was
	// a problem with the graceful shutdown and we return the error.
	err = <-shutdownError
	if err != nil {
		return err
	}

	// At this point we know that the graceful shutdown completed successfully and
	// we log a "stopped server" message.
	app.logger.Info("Stopped server.", "addr", server.Addr)

	return nil
}
