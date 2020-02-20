package gopher

import (
	"errors"
	"fmt"
	"strings"
)

var ErrStatus = errors.New("status")

type Error struct {
	Status     Status
	URL        URL
	Message    string
	Confidence float64
	Raw        []byte
}

func NewError(u URL, status Status, msg string, confidence float64) *Error {
	if confidence < 0 {
		confidence = 0
	} else if confidence > 1 {
		confidence = 1
	}
	return &Error{
		URL:        u,
		Status:     status,
		Message:    msg,
		Confidence: confidence,
	}
}

func (e *Error) Is(err error) bool {
	return e.Status == err || err == ErrStatus
}

func (e *Error) Error() string {
	return fmt.Sprintf("gopher: request failed with status %d: %s", e.Status, strings.TrimSpace(e.Message))
}
