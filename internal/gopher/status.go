package gopher

import "fmt"

// Uses GopherII codes wherever possible:
// https://tools.ietf.org/html/draft-matavka-gopher-ii-02#section-9.1
//
// Wherever not possible, uses 6xx for client or 7xx for server errors.
// 6xx and 7xx codes are subject to change outside the GopherII RFC.
// This is discouraged by the RFC and will be minimised.
type Status int

const (
	OK Status = 0

	StatusBadRequest     Status = 400
	StatusUnauthorized   Status = 401
	StatusForbidden      Status = 403
	StatusNotFound       Status = 404
	StatusRequestTimeout Status = 408
	StatusGone           Status = 410
	StatusInternal       Status = 500
	StatusNotImplemented Status = 501
	StatusUnavailable    Status = 503

	StatusGeneralError Status = 600 // Non-specific error code
	StatusEmpty        Status = 601
)

// Error is implemented to allow the Error struct to match against
// a Status using errors.Is.
func (s Status) Error() string {
	return fmt.Sprintf("%d", s)
}
