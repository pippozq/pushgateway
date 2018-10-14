package errors

import (
	"net/http"
)

//go:generate courier gen error StatusError
type StatusError int

func (StatusError) ServiceCode() int {
	return 999 * 1e3
}

const (
	// InternalServerError
	// Something wrong in server
	InternalServerError StatusError = http.StatusInternalServerError*1e6 + iota + 1
)

const (
	// @errTalk BadRequest
	BadRequest StatusError = http.StatusBadRequest*1e6 + iota + 1
)

const (
	// Not Found
	NotFound StatusError = http.StatusNotFound*1e6 + iota + 1
	// @errTalk MetricNotFound
	MetricNotFound
)
