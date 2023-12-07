package data

import (
	"fmt"
	"strconv"
)

type Runtime int32

// MarshalJSON returns a string in the format "<runtime> mins". Implement a
// MarshalJSON() method on Runtime type so that it satisfies the json.Marshaler
// interface.
func (r Runtime) MarshalJSON() ([]byte, error) {
	jsonValue := fmt.Sprintf("%d mins", r)
	quotedJSONValue := strconv.Quote(jsonValue)
	return []byte(quotedJSONValue), nil
}
