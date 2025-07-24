package response

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// Response represents the standard API response format
type Response struct {
	Success   bool        `json:"success"`
	Message   string      `json:"message,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Error     *ErrorInfo  `json:"error,omitempty"`
	Meta      *Meta       `json:"meta,omitempty"`
	Timestamp string      `json:"timestamp"`
}

// ErrorInfo provides detailed error information
type ErrorInfo struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// Meta provides additional metadata for the response
type Meta struct {
	Page       int `json:"page,omitempty"`
	Limit      int `json:"limit,omitempty"`
	Total      int `json:"total,omitempty"`
	TotalPages int `json:"total_pages,omitempty"`
}

// ValidationError represents field validation errors
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Value   string `json:"value,omitempty"`
}

// JSON sends a successful JSON response
func JSON(w http.ResponseWriter, data interface{}, statusCode int) {
	response := Response{
		Success:   true,
		Data:      data,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	sendJSONResponse(w, response, statusCode)
}

// JSONWithMessage sends a successful JSON response with a custom message
func JSONWithMessage(w http.ResponseWriter, data interface{}, message string, statusCode int) {
	response := Response{
		Success:   true,
		Message:   message,
		Data:      data,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	sendJSONResponse(w, response, statusCode)
}

// JSONWithMeta sends a successful JSON response with metadata (useful for pagination)
func JSONWithMeta(w http.ResponseWriter, data interface{}, meta *Meta, statusCode int) {
	response := Response{
		Success:   true,
		Data:      data,
		Meta:      meta,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	sendJSONResponse(w, response, statusCode)
}

// Error sends an error JSON response
func Error(w http.ResponseWriter, message string, statusCode int) {
	response := Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    http.StatusText(statusCode),
			Message: message,
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	sendJSONResponse(w, response, statusCode)
}

// ErrorWithCode sends an error JSON response with a custom error code
func ErrorWithCode(w http.ResponseWriter, code, message string, statusCode int) {
	response := Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	sendJSONResponse(w, response, statusCode)
}

// ErrorWithDetails sends an error JSON response with additional details
func ErrorWithDetails(w http.ResponseWriter, code, message string, details interface{}, statusCode int) {
	response := Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
			Details: details,
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	sendJSONResponse(w, response, statusCode)
}

// ValidationError sends a validation error response
func ValidationErrors(w http.ResponseWriter, errors []ValidationError) {
	response := Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    "VALIDATION_ERROR",
			Message: "Validation failed",
			Details: errors,
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	sendJSONResponse(w, response, http.StatusBadRequest)
}

// InternalServerError sends a generic internal server error
func InternalServerError(w http.ResponseWriter) {
	Error(w, "An internal server error occurred", http.StatusInternalServerError)
}

// NotFound sends a not found error
func NotFound(w http.ResponseWriter, resource string) {
	message := "Resource not found"
	if resource != "" {
		message = fmt.Sprintf("%s not found", resource)
	}
	Error(w, message, http.StatusNotFound)
}

// Unauthorized sends an unauthorized error
func Unauthorized(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Authentication required"
	}
	Error(w, message, http.StatusUnauthorized)
}

// Forbidden sends a forbidden error
func Forbidden(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Access forbidden"
	}
	Error(w, message, http.StatusForbidden)
}

// BadRequest sends a bad request error
func BadRequest(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Bad request"
	}
	Error(w, message, http.StatusBadRequest)
}

// TooManyRequests sends a rate limit exceeded error
func TooManyRequests(w http.ResponseWriter) {
	Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
}

// sendJSONResponse is a helper function that actually sends the JSON response
func sendJSONResponse(w http.ResponseWriter, response Response, statusCode int) {
	// Set response headers
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	
	// Set status code
	w.WriteHeader(statusCode)

	// Encode and send response
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ") // Pretty print in development
	
	if err := encoder.Encode(response); err != nil {
		// If JSON encoding fails, send a basic error response
		log.Printf("Failed to encode JSON response: %v", err)
		
		// Clear any previous headers and content
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// NewMeta creates a new Meta struct for pagination
func NewMeta(page, limit, total int) *Meta {
	totalPages := (total + limit - 1) / limit // Ceiling division
	
	return &Meta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}
}

// NewValidationError creates a new ValidationError
func NewValidationError(field, message, value string) ValidationError {
	return ValidationError{
		Field:   field,
		Message: message,
		Value:   value,
	}
}

// Common error codes constants
const (
	ErrorCodeValidation      = "VALIDATION_ERROR"
	ErrorCodeNotFound        = "NOT_FOUND"
	ErrorCodeUnauthorized    = "UNAUTHORIZED"
	ErrorCodeForbidden       = "FORBIDDEN"
	ErrorCodeRateLimit       = "RATE_LIMIT_EXCEEDED"
	ErrorCodeInternalServer  = "INTERNAL_SERVER_ERROR"
	ErrorCodeBadRequest      = "BAD_REQUEST"
	ErrorCodeConflict        = "CONFLICT"
	ErrorCodeUnsupportedType = "UNSUPPORTED_TYPE"
)

// Success response helpers

// Created sends a 201 Created response
func Created(w http.ResponseWriter, data interface{}, message string) {
	if message == "" {
		message = "Resource created successfully"
	}
	JSONWithMessage(w, data, message, http.StatusCreated)
}

// Updated sends a 200 OK response for updates
func Updated(w http.ResponseWriter, data interface{}, message string) {
	if message == "" {
		message = "Resource updated successfully"
	}
	JSONWithMessage(w, data, message, http.StatusOK)
}

// Deleted sends a 200 OK response for deletions
func Deleted(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Resource deleted successfully"
	}
	JSONWithMessage(w, nil, message, http.StatusOK)
}

// NoContent sends a 204 No Content response
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}