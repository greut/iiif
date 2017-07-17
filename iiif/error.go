package iiif

import (
	"fmt"
	"net/http"
)

// HTTPError represents a HTTP error to be shown to the user.
type HTTPError struct {
	StatusCode int
	Message    string
}

// Error formats the HTTPError message.
func (e HTTPError) Error() string {
	return fmt.Sprintf("%d (%s) %s", e.StatusCode, http.StatusText(e.StatusCode), e.Message)
}
