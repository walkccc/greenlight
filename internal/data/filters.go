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

func (f Filters) limit() int {
	return f.PageSize
}

func (f Filters) offset() int {
	return (f.Page - 1) * f.PageSize
}

func ValidateFilters(v *validator.Validator, f Filters) {
	v.Check(f.Page > 0, "page", "must be greater than zero")
	v.Check(f.Page <= 10_000_000, "page", "must be a maximum of 10 million")
	v.Check(f.PageSize > 0, "page_size", "must be greater than 0")
	v.Check(f.PageSize <= 100, "page_size", "must be a maximum of 100")
	v.Check(validator.PermittedValue(f.Sort, f.SortSafeValues...), "sort", "invalid sort value")
}

type Metadata struct {
	CurrentPage  int `json:"current_page,omitempty"`
	PageSize     int `json:"page_size,omitempty"`
	FirstPage    int `json:"first_page,omitempty"`
	LastPage     int `json:"last_page,omitempty"`
	TotalRecords int `json:"total_records,omitempty"`
}

// calculateMetadata calculates the appropriate pagination metadata values given
// the total number of records, current page, and page size values. For example,
// if there were 12 records in total and a page size of 5, the last page value
// will be (12 - 1) / 5 + 1 = 3.
func calculateMetadata(totalRecords, page, pageSize int) Metadata {
	if totalRecords == 0 {
		return Metadata{}
	}
	return Metadata{
		CurrentPage:  page,
		PageSize:     pageSize,
		FirstPage:    1,
		LastPage:     (totalRecords-1)/pageSize + 1,
		TotalRecords: totalRecords,
	}
}
