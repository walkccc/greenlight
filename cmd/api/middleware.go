package main

import (
	"errors"
	"expvar"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tomasen/realip"
	"github.com/walkccc/greenlight/internal/data"
	"github.com/walkccc/greenlight/internal/validator"
	"golang.org/x/time/rate"
)

func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create a deferred function (which will always be run in the event of a
		// panic as Go unwinds the stack).
		defer func() {
			// Use the built-in recover function to check if there has been a panic.
			if err := recover(); err != nil {
				// If there was a panic, set a "Connection: close" header on the
				// response. This acts as a trigger to make Go's HTTP server
				// automatically close the current connection after a response has been
				// sent.
				w.Header().Set("Connection", "close")
				// The value returned by recover() has the type any, so we use
				// fmt.Errorf() to normalize it into an error and call our
				// serverErrorResponse() helper. In turn, this will log the error at the
				// ERROR level and sent the client a 500 Internal Server Error response.
				app.serverErrorResponse(w, r, fmt.Errorf("%s", err))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func (app *application) rateLimit(next http.Handler) http.Handler {
	type client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}

	// A mutex and a map to hold the clients' IP addresses and clients.
	var (
		mtx     sync.Mutex
		clients = make(map[string]*client)
	)

	// A background goroutine which removes old entries from the clients map once
	// every minute.
	go func() {
		for {
			time.Sleep(time.Minute)

			// Lock the mutex to prevent any rate limiter checks from happening while
			// the cleanup is taking place.
			mtx.Lock()

			// Loop through all clients. If they haven't been seen within the last
			// three minutes, delete the corresponding entry from the map.
			for ip, client := range clients {
				if time.Since(client.lastSeen) > 3*time.Minute {
					delete(clients, ip)
				}
			}

			// Importantly, unlock the mutex when the cleanup is complete.
			mtx.Unlock()
		}
	}()

	// The function we're returning is a closure, which 'closes over' the limiter
	// variable.
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if app.config.limiter.enabled {
			// Retrieve the client IP address from any X-Forwarded-For or
			// X-Real-IP headers, falling back to use r.RemoteAddr if neither of them
			// are present.
			ip := realip.FromRequest(r)

			// Lock the mutex to prevent this code from being executed concurrently.
			mtx.Lock()

			if _, found := clients[ip]; !found {
				// Initialize a new rate limiter which allows an average of 2 requests
				// per second, with a maximum of 4 requests in a single 'burst'.
				clients[ip] = &client{
					limiter: rate.NewLimiter(
						rate.Limit(app.config.limiter.rps),
						app.config.limiter.burst,
					),
				}
			}

			clients[ip].lastSeen = time.Now()

			if !clients[ip].limiter.Allow() {
				mtx.Unlock()
				app.rateLimitExceededResponse(w, r)
				return
			}

			// Very importantly, unlock the mutex before calling the next handler in
			// the chain. Notice that we DON'T use defer to unlock the mutex, as that
			// would mean that the mutex isn't unlocked until all the handlers
			// downstream of this middleware have also returned.
			mtx.Unlock()
		}

		next.ServeHTTP(w, r)
	})
}

// authenticate tries to retrieve the value of the Authorization header from the
// request.
//   - If a valid authentication token is provided in the Authorization header,
//     then a User struct containing the corresponding user details will be
//     stored in the request context.
//   - If no Authorization header is provided at all, our AnonymousUser struct
//     will be stored in the request context.
//   - If the Authorization header is provided, but it's malformed or contains
//     an invalid value, the client will be sent a 401 Unauthorized response
//     using invalidAuthenticationTokenRespons().
func (app *application) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This indicates to any caches that the response may vary based on the
		// value of the Authorization header in the request.
		w.Header().Add("Vary", "Authorization")

		// Retrieve the value of the Authorization header from the request. This
		// will return the empty string "" if there is no such header found.
		authorizationHeader := r.Header.Get("Authorization")
		if authorizationHeader == "" {
			r = app.contextSetUser(r, data.AnonymousUser)
			next.ServeHTTP(w, r)
			return
		}

		// We expect the value of the Authorization header to be in the format
		// "Bearer <token>". We try to split this into its constituent parts, and if
		// the header isn't in the expected format, we return a 401 Unauthorized
		// reponse using invalidAuthenticationTokenResponse().
		headerParts := strings.Split(authorizationHeader, " ")
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}

		token := headerParts[1]
		v := validator.New()
		if data.ValidateTokenPlaintext(v, token); !v.Valid() {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}

		// Retrieve the details of the user associated with the authentication
		// token. Note that we are using ScopeAuthentication as the first parameter
		// here.
		user, err := app.models.Users.GetForToken(data.ScopeAuthentication, token)
		if err != nil {
			switch {
			case errors.Is(err, data.ErrRecordNotFound):
				app.invalidAuthenticationTokenResponse(w, r)
			default:
				app.serverErrorResponse(w, r, err)
			}
			return
		}

		r = app.contextSetUser(r, user)
		next.ServeHTTP(w, r)
	})
}

// requireAuthenticatedUser checks that a user is not anonymous.
func (app *application) requireAuthenticatedUser(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := app.contextGetUser(r)

		if user.IsAnonymous() {
			app.authenticationRequiredResponse(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// requireActivatedUser checks that a user is both authenticated and activated.
func (app *application) requireActivatedUser(next http.HandlerFunc) http.HandlerFunc {
	fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := app.contextGetUser(r)

		if !user.Activated {
			app.inactiveAccountResponse(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})

	return app.requireAuthenticatedUser(fn)
}

// requirePermissions checks that a user has the required permission code.
func (app *application) requirePermission(code string, next http.HandlerFunc) http.HandlerFunc {
	fn := func(w http.ResponseWriter, r *http.Request) {
		user := app.contextGetUser(r)

		permissions, err := app.models.Permissions.GetAllForUser(user.ID)
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}

		if !permissions.Include(code) {
			app.notPermittedResponse(w, r)
			return
		}

		next.ServeHTTP(w, r)
	}

	return app.requireActivatedUser(fn)
}

func (app *application) enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Origin")
		w.Header().Add("Vary", "Access-Control-Request-Method")

		origin := r.Header.Get("Origin")
		if origin != "" {
			for _, trustedOrigin := range app.config.cors.trustedOrigins {
				if origin == trustedOrigin {
					w.Header().Set("Access-Control-Allow-Origin", origin)

					// Treat it as a preflight request.
					if r.Method == http.MethodOptions &&
						r.Header.Get("Access-Control-Request-Method") != "" {
						// Set the necessary preflight response headers.
						w.Header().Set("Access-Control-Allow-Methods", "OPTIONS, PUT, PATCH, DELETE")
						w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")

						// Return from the middleware with no further action.
						w.WriteHeader(http.StatusOK)
						return
					}

					break
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

// metricsResponseWriter wraps an existing http.ResponseWriter and also contains
// a field for recording the response status code, and a boolean flag to
// indicate whether the response headers have already been written.
type metricsResponseWriter struct {
	wrapped       http.ResponseWriter
	statusCode    int
	headerWritten bool
}

// newMetricsResponseWriter returns a new metricsResponseWriter instance which
// wraps a given http.ResponseWriter and has a status code of 200 (which is the
// status code that Go will send in a HTTP response by default).
func newMetricsResponseWriter(w http.ResponseWriter) *metricsResponseWriter {
	return &metricsResponseWriter{
		wrapped:    w,
		statusCode: http.StatusOK,
	}
}

func (mw *metricsResponseWriter) Header() http.Header {
	return mw.wrapped.Header()
}

func (mw *metricsResponseWriter) WriteHeader(statusCode int) {
	mw.wrapped.WriteHeader(statusCode)
	if !mw.headerWritten {
		mw.statusCode = statusCode
		mw.headerWritten = true
	}
}

func (mw *metricsResponseWriter) Write(b []byte) (int, error) {
	mw.headerWritten = true
	return mw.wrapped.Write(b)
}

func (mw *metricsResponseWriter) Unwrap() http.ResponseWriter {
	return mw.wrapped
}

func (app *application) metrics(next http.Handler) http.Handler {
	// Initialize the new expvar variables when the middleware chain is first
	// built.
	var (
		totalRequestsReceived           = expvar.NewInt("total_requests_received")
		totalResponsesSent              = expvar.NewInt("total_responses_sent")
		totalProcessingTimeMicroseconds = expvar.NewInt("total_processing_time_μs")
		totalResponsesSentByStatus      = expvar.NewMap("total_responses_sent_by_status")
	)

	// The following code will be run for every request.
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Record the time that we started to process the request.
		start := time.Now()

		// Use the Add() method to increment the number of requests received by 1.
		totalRequestsReceived.Add(1)

		// Create a new metricsResponseWriter, which wraps the original
		// http.ResponseWriter value that the metrics middleware received.
		mw := newMetricsResponseWriter(w)

		// Call the next handler in the chain using the new metricsResponseWriter as
		// the http.ResponseWriter value.
		next.ServeHTTP(mw, r)

		// On the way back up the middleware chain, increment the number of
		// responses sent by 1.
		totalResponsesSent.Add(1)

		// Calculate the number of microseconds since we began to process the
		// request, then increment the total processing time by this amount.
		// At this point, the response status code should be stored in the
		// mw.statusCode field. Note that the expvar map is string-keyed, so we need
		// to use the strconv.Itoa() function to convert the status code (which is
		// an integer) to a string. Then we use the Add() method on our new
		// totalResponsesSentByStatus map to increment the count for the given
		// status code by 1.
		totalResponsesSentByStatus.Add(strconv.Itoa(mw.statusCode), 1)

		// Calculate the number of microseconds since we began to process the
		// request, then increment the total processing time by this amount.
		duration := time.Since(start).Microseconds()
		totalProcessingTimeMicroseconds.Add(duration)
	})
}
