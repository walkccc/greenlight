package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/julienschmidt/httprouter"
	"github.com/walkccc/greenlight/internal/validator"
)

// readIDParam retrieves the "id" URL parameter from the current request
// context, then converts it to an integer and returns it. If the operation
// isn't successful, return 0 and an error.
func (app *application) readIDParam(r *http.Request) (int64, error) {
	params := httprouter.ParamsFromContext(r.Context())

	id, err := strconv.ParseInt(params.ByName("id"), 10, 64)
	if err != nil || id < 1 {
		return 0, errors.New("invalid id parameter")
	}

	return id, nil
}

type envelope map[string]any

// writeJSON takes the destination http.ResponseWriter, the HTTP status code to
// send, the data to encode to JSON, and a header map containing any additional
// HTTP headers we want to include in the response.
func (app *application) writeJSON(
	w http.ResponseWriter,
	statusCode int,
	data envelope,
	headers http.Header,
) error {
	js, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		return err
	}

	js = append(js, '\n')

	for key, value := range headers {
		w.Header()[key] = value
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_, err = w.Write(js)
	return err
}

// readJSON decodes the JSON from the request body, then triage the errors and
// replace them with the custom messages as necessary.
func (app *application) readJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	// Use http.MaxBytesReader() to limit the size of the request body to 1MB.
	maxBytes := 1_048_576
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	err := decoder.Decode(dst)
	if err != nil {
		// If there's an error during decoding, start the triage.
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidUnmarshalError *json.InvalidUnmarshalError
		var maxBytesError *http.MaxBytesError

		switch {
		// Check whether the error has the type *json.SyntaxError.
		case errors.As(err, &syntaxError):
			return fmt.Errorf(
				"body contains badly-formed JSON (at character %d)",
				syntaxError.Offset,
			)

		// In some circumstances, Decode() may also return an io.ErrUnexpectedEOF
		// error for syntax errors in the JSON.
		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New("body contains badly-formed JSON")

		// This error occurs when the JSON value is the wrong type for the target
		// destination. If the error relates to a specific field, then we include
		// that in our error message to make it easier for the client to debug.
		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf(
					"body contains incorrect JSON type for field %q",
					unmarshalTypeError.Field,
				)
			}
			return fmt.Errorf(
				"body contains incorrect JSON type (at character %d)",
				unmarshalTypeError.Offset,
			)

		// An io.EOF error will be returned by Decode() if the request body is
		// empty.
		case errors.Is(err, io.EOF):
			return errors.New("body must not be empty")

		// If the JSON contains a field which cannot be mapped to the target
		// destination then Decode() will now return an error message in the format
		// "json: unknown field "<name>"". We check for this, extract the field name
		// from the error, and interpolate it into our custom error message.
		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			return fmt.Errorf("body contains unknown key %s", fieldName)

		// Check whether the error has type *http.MaxBytesError.
		case errors.As(err, &maxBytesError):
			return fmt.Errorf("ody must not be larger than %d bytes", maxBytesError.Limit)

		// A json.InvalidUnmarshalError error will be returned if we pass something
		// that is not a non-nil pointer to Decode(). We catch this and panic(),
		// rather than return an error to our handler.
		case errors.As(err, &invalidUnmarshalError):
			panic(err)

		// Return the error message as-is.
		default:
			return err
		}
	}

	// Call Decode() again, using a pointer to an empty anonymous struct as the
	// destination. If the request body only contained a single JSON value this
	// will return an io.EOF error. So if we get anything else, we know that there
	// is additional data in the request body and we return our own custom error
	// message.
	err = decoder.Decode(&struct{}{})
	if !errors.Is(err, io.EOF) {
		return errors.New("body must only contain a single JSON value")
	}

	return nil
}

// readString returns a string value from the query string. If no matching key
// can be found, it returns the default value.
func (app *application) readString(qs url.Values, key string, defaultValue string) string {
	s := qs.Get(key)
	if s == "" {
		return defaultValue
	}
	return s
}

// readCSV reads a string value from the query string and then splits it into a
// slice on the comma character. If no matching key can be found, it returns the
// default value.
func (app *application) readCSV(qs url.Values, key string, defaultValue []string) []string {
	csv := qs.Get(key)
	if csv == "" {
		return defaultValue
	}
	return strings.Split(csv, ",")
}

// readInt reads a string value from the query string and converts it to an
// integer. If no matching key can be found, it returns the default value. If
// the value can't be converted to an integer, then it records an error message
// in the provided Validator instance.
func (app *application) readInt(
	qs url.Values,
	key string,
	defaultValue int,
	v *validator.Validator,
) int {
	s := qs.Get(key)
	if s == "" {
		return defaultValue
	}

	i, err := strconv.Atoi(s)
	if err != nil {
		v.AddError(key, "must be an integer value")
		return defaultValue
	}

	return i
}

// background accepts an arbitrary function as a parameter and launches a
// background goroutine that is capable of recovering from any panics that may
// occur.
func (app *application) background(fn func()) {
	go func() {
		// Recover any panic.
		defer func() {
			if err := recover(); err != nil {
				app.logger.Error(fmt.Sprintf("%v", err))
			}
		}()

		fn()
	}()
}
