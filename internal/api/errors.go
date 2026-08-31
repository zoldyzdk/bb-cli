package api

import (
	"errors"
	"fmt"
)

type HTTPError struct {
	StatusCode int
	Message    string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("API error (%d): %s", e.StatusCode, e.Message)
}

func IsForbidden(err error) bool {
	var he *HTTPError
	return errors.As(err, &he) && he.StatusCode == 403
}

func IsUnauthorized(err error) bool {
	var he *HTTPError
	return errors.As(err, &he) && he.StatusCode == 401
}
