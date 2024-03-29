package main

import (
	"fmt"
	"net/http"
)

// logError is a generic helper for logging an error message along with the
// current request method and URL as attributes in the log entry.
func (app *application) logError(r *http.Request, err error) {
	var (
		method = r.Method
		uri    = r.URL.RequestURI()
	)
	app.logger.Error(err.Error(), "method", method, "uri", uri)
}

// errorResponse method is a generic helper for sending JSON-formatted error
// messages to the client with a given status code. Note that we're using the
// any type for the message parameter, rather than just a string type, as this
// gives us more flexibility over the values that we can include in the
// response.
func (app *application) errorResponse(
	w http.ResponseWriter,
	r *http.Request,
	statusCode int,
	message any,
) {
	env := envelope{"error": message}

	err := app.writeJSON(w, statusCode, env, nil)
	if err != nil {
		app.logError(r, err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

// serverErrorResponse logs the detailed error message when our application
// encounters an unexpected problem at runtime. It uses the errorResponse()
// helper to send a 500 Internal Server Error status code and JSON response
// (containing a generic error message) to the client.
func (app *application) serverErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.logError(r, err)

	message := "The server encountered a problem and could not process your request."
	app.errorResponse(w, r, http.StatusInternalServerError, message)
}

// notFoundResponse sends a 404 Not Found status code and JSON response to the
// client.
func (app *application) notFoundResponse(w http.ResponseWriter, r *http.Request) {
	message := "The requested resource could not be found."
	app.errorResponse(w, r, http.StatusNotFound, message)
}

// methodNotAllowedResponse sends a 405 Method Not Allowed status code and JSON
// response to the client.
func (app *application) methodNotAllowedResponse(w http.ResponseWriter, r *http.Request) {
	message := fmt.Sprintf("The %s method is not supported for this resource.", r.Method)
	app.errorResponse(w, r, http.StatusMethodNotAllowed, message)
}

// badRequestResposne sends a 400 Bad Request status code and JSON response to
// the client.
func (app *application) badRequestResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.errorResponse(w, r, http.StatusBadRequest, err.Error())
}

// failedValidationResponse sends a 422 Unprocessable Entity status code and
// JSON response to the client. Note that the errors has the same type as
// Validator.Errors.
func (app *application) failedValidationResponse(
	w http.ResponseWriter,
	r *http.Request,
	errors map[string]string,
) {
	app.errorResponse(w, r, http.StatusUnprocessableEntity, errors)
}

// editConflictResponse sends a 409 Conflict status code and JSON response to
// the client.
func (app *application) editConflictResponse(w http.ResponseWriter, r *http.Request) {
	message := "Unable to update the record due to an edit conflict, please try again."
	app.errorResponse(w, r, http.StatusConflict, message)
}

// rateLimitExceededResponse sends a 429 Too Many Requests status code and JSON
// response to the client.
func (app *application) rateLimitExceededResponse(w http.ResponseWriter, r *http.Request) {
	message := "rate limit exceeded"
	app.errorResponse(w, r, http.StatusTooManyRequests, message)
}

// invalidCredentialsResponse sends a 401 Unauthorized status code and JSON
// response to the client.
func (app *application) invalidCredentialsResponse(w http.ResponseWriter, r *http.Request) {
	message := "invalid authentication credentials"
	app.errorResponse(w, r, http.StatusUnauthorized, message)
}

// invalidAuthenticationTokenResponse sends a 401 Unauthorized status code and
// JSON response to the client.
func (app *application) invalidAuthenticationTokenResponse(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	message := "invalid or missing authentication token"
	app.errorResponse(w, r, http.StatusUnauthorized, message)
}

// authenticationRequiredResponse sends a 401 Unauthorized status code and JSON
// response to the client.
func (app *application) authenticationRequiredResponse(w http.ResponseWriter, r *http.Request) {
	message := "You must be authenticated to access this resource."
	app.errorResponse(w, r, http.StatusUnauthorized, message)
}

// inactiveAccountResponse sends a 403 Forbidden status code and JSON response
// to the client.
func (app *application) inactiveAccountResponse(w http.ResponseWriter, r *http.Request) {
	message := "Your user account must be activated to access this resource."
	app.errorResponse(w, r, http.StatusForbidden, message)
}

// notPermittedResponse sends a 403 Forbidden status code and JSON response to
// the client.
func (app *application) notPermittedResponse(w http.ResponseWriter, r *http.Request) {
	message := "Your user account doesn't have the necessary permissions to access this resource."
	app.errorResponse(w, r, http.StatusForbidden, message)
}
