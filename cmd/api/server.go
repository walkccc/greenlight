package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// serve starts a server. When we receive a SIGINT or SIGTERM signal, we instruct our server to stop
// accepting any new HTTP requests, and give any in-flight requests a 'grace period' of 30 seconds
// to complete before the application is terminated.
func (app *application) serve() error {
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", app.config.port),
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// shutdownError is a channel that receives any errors returned by the graceful Showtdown().
	shutdownError := make(chan error)

	go func() {
		// Intercept the signals.
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		s := <-quit

		app.logger.PrintInfo("shutting down server", map[string]string{
			"signal": s.String(),
		})

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Call Shutdown() on our server, passing in the context. Shutdown() will return nil if the
		// graceful shutdown was successful, or an error (which may happen because of a problem
		// closing the listeners, or because the shutdown didn't complete before the 30-second
		// context deadline is hit). We relay this return value to the shutdownError channel.
		err := server.Shutdown(ctx)
		if err != nil {
			shutdownError <- err
		}

		app.logger.PrintInfo("completing background tasks", map[string]string{
			"addr": server.Addr,
		})

		// Call Wait() to block until our WaitGroup counter reaches zero -- essentially blocking
		// until the background goroutines have finished. Then, we return nil on the shutdownError
		// channel to indicate that the shutdown completed without any issues.
		app.wg.Wait()
		shutdownError <- nil
	}()

	app.logger.PrintInfo("starting server", map[string]string{
		"env":  app.config.env,
		"addr": server.Addr,
	})

	// Calling Shutdown() on our server will cause ListenAndServe() to immediately return a
	// http.ErrServerClosed error. So if we see this error, it's actually a good thing and an
	// indication that the graceful shutdown has started. So we check specifically for this, only
	// returning error if it's NOT http.ErrServerClosed.
	err := server.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	// Otherwise, we wait to receive the return value from Shutdown() on the sutdownError channel.
	// If return value is an error, we know that there was a problem with the graceful shutdown and
	// we return the error.
	err = <-shutdownError
	if err != nil {
		return err
	}

	// At this point, we know that the graceful shutdown completed successfully and we log a
	// "stopped server" message.
	app.logger.PrintInfo("stopped server", map[string]string{
		"addr": server.Addr,
	})

	return nil
}
