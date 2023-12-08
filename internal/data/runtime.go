package data

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ErrInvalidRuntimeFormat is an error that UnmarshalJSON() can return if we're
// unable to parse or convert the JSON string.
var ErrInvalidRuntimeFormat = errors.New("invalid runtime format")

type Runtime int32

// MarshalJSON returns a string in the format "<runtime> mins". Implement a
// MarshalJSON() method on Runtime type so that it satisfies the json.Marshaler
// interface.
func (r Runtime) MarshalJSON() ([]byte, error) {
	jsonValue := fmt.Sprintf("%d mins", r)
	quotedJSONValue := strconv.Quote(jsonValue)
	return []byte(quotedJSONValue), nil
}

// UnmarshalJSON ensures that Runtime satisfies the json.Unmarshaler interface.
// IMPORTANT: because UnmarshalJSON() needs to modify the receiver (our Runtime
// type), we must use a pointer receiver for this to work correctly. Otherwise,
// we will only be modifying a copy (which is then discarded when this method
// returns).
func (r *Runtime) UnmarshalJSON(jsonValue []byte) error {
	unquotedJSONValue, err := strconv.Unquote(string(jsonValue))
	if err != nil {
		return ErrInvalidRuntimeFormat
	}

	parts := strings.Split(unquotedJSONValue, " ")
	if len(parts) != 2 || parts[1] != "mins" {
		return ErrInvalidRuntimeFormat
	}

	num, err := strconv.ParseInt(parts[0], 10, 32)
	if err != nil {
		return ErrInvalidRuntimeFormat
	}

	*r = Runtime(num)
	return nil
}
