package data

import (
	"strings"

	"github.com/walkccc/greenlight/internal/validator"
)

type Filters struct {
	Page           int
	PageSize       int
	Sort           string
	SortSafeValues []string
}

// sortColumn extracts the column name from the Sort field if it matches one of
// the entries in SortSafeValues.
func (f Filters) sortColumn() string {
	for _, sortSafeValue := range f.SortSafeValues {
		if f.Sort == sortSafeValue {
			return strings.TrimPrefix(f.Sort, "-")
		}
	}

	// A sensible failsafe to help stop a SQL injection attack.
	panic("unsafe sort parameter: " + f.Sort)
}

// sortDirection returns the sort direction ("ASC" or "DESC") depending on the
// prefix character of Sort field.
func (f Filters) sortDirection() string {
	if strings.HasPrefix(f.Sort, "-") {
		return "DESC"
	}
	return "ASC"
}

func ValidateFilters(v *validator.Validator, f Filters) {
	v.Check(f.Page > 0, "page", "must be greater than zero")
	v.Check(f.Page <= 10_000_000, "page", "must be a maximum of 10 million")
	v.Check(f.PageSize > 0, "page_size", "must be greater than 0")
	v.Check(f.PageSize <= 100, "page_size", "must be a maximum of 100")
	v.Check(validator.PermittedValue(f.Sort, f.SortSafeValues...), "sort", "invalid sort value")
}
