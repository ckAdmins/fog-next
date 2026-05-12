// Package response provides helpers for writing consistent JSON API responses.
package response

import (
	"encoding/json"
	"net/http"
)

// JSON writes a JSON response with the given status code and body.
func JSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// OK writes a 200 JSON response.
func OK(w http.ResponseWriter, body any) {
	JSON(w, http.StatusOK, body)
}

// Created writes a 201 JSON response.
func Created(w http.ResponseWriter, body any) {
	JSON(w, http.StatusCreated, body)
}

// NoContent writes a 204 response.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// Error writes a Problem Details (RFC 7807) JSON error response.
func Error(w http.ResponseWriter, status int, title, detail string) {
	JSON(w, status, map[string]any{
		"status": status,
		"title":  title,
		"detail": detail,
	})
}

// BadRequest is a shorthand for a 400 error.
func BadRequest(w http.ResponseWriter, detail string) {
	Error(w, http.StatusBadRequest, "Bad Request", detail)
}

// Unauthorized writes a 401 error.
func Unauthorized(w http.ResponseWriter) {
	Error(w, http.StatusUnauthorized, "Unauthorized", "Authentication required")
}

// Forbidden writes a 403 error.
func Forbidden(w http.ResponseWriter) {
	Error(w, http.StatusForbidden, "Forbidden", "Insufficient permissions")
}

// NotFound writes a 404 error.
func NotFound(w http.ResponseWriter, resource string) {
	Error(w, http.StatusNotFound, "Not Found", resource+" not found")
}

// Conflict writes a 409 error.
func Conflict(w http.ResponseWriter, detail string) {
	Error(w, http.StatusConflict, "Conflict", detail)
}

// InternalError writes a 500 error without leaking internal details.
func InternalError(w http.ResponseWriter) {
	Error(w, http.StatusInternalServerError, "Internal Server Error", "An unexpected error occurred")
}

// Decode decodes a JSON request body into dst, writing a 400 on failure.
// Returns true on success.
func Decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		BadRequest(w, "invalid JSON: "+err.Error())
		return false
	}
	return true
}

// ListResponse is the standard envelope for collection endpoints.
type ListResponse[T any] struct {
	Data []T `json:"data"`
}

// ListOf wraps items in a ListResponse, normalising nil slices to an empty array.
func ListOf[T any](items []T) ListResponse[T] {
	if items == nil {
		items = make([]T, 0)
	}
	return ListResponse[T]{Data: items}
}
