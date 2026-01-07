package api

import (
	"errors"
	"fmt"
)

var (
	// ErrUnauthorized indicates authentication failed
	ErrUnauthorized = errors.New("authentication failed")
	// ErrNotFound indicates repository or tag not found
	ErrNotFound = errors.New("repository or tag not found")
	// ErrRateLimited indicates rate limit exceeded
	ErrRateLimited = errors.New("rate limit exceeded")
	// ErrNetworkError indicates a network error occurred
	ErrNetworkError = errors.New("network error")
	// ErrInvalidResponse indicates invalid API response
	ErrInvalidResponse = errors.New("invalid API response")
)

// APIError represents an error from the Docker Hub API
type APIError struct {
	StatusCode int
	Message    string
	Endpoint   string
}

// Error implements the error interface
func (e *APIError) Error() string {
	return fmt.Sprintf("API error (status %d) at %s: %s", e.StatusCode, e.Endpoint, e.Message)
}

// NewAPIError creates a new APIError
func NewAPIError(statusCode int, endpoint, message string) *APIError {
	return &APIError{
		StatusCode: statusCode,
		Message:    message,
		Endpoint:   endpoint,
	}
}
