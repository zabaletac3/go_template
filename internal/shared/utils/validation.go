// internal/shared/utils/validation.go
package utils

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// Email validation regex pattern
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// IsValidEmail validates email format
func IsValidEmail(email string) bool {
	email = strings.TrimSpace(email)
	if len(email) == 0 || len(email) > 255 {
		return false
	}
	return emailRegex.MatchString(email)
}

// IsValidUsername validates username format
func IsValidUsername(username string) bool {
	username = strings.TrimSpace(username)
	
	// Check length
	if len(username) < 3 || len(username) > 30 {
		return false
	}
	
	// Check if username contains only allowed characters
	for _, char := range username {
		if !unicode.IsLetter(char) && !unicode.IsDigit(char) && char != '_' {
			return false
		}
	}
	
	return true
}

// IsValidPassword validates password strength
func IsValidPassword(password string) bool {
	if len(password) < 8 || len(password) > 128 {
		return false
	}
	
	var (
		hasUpper   bool
		hasLower   bool
		hasDigit   bool
		hasSpecial bool
	)
	
	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasDigit = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}
	
	// Require at least 3 of the 4 character types
	checkCount := 0
	if hasUpper {
		checkCount++
	}
	if hasLower {
		checkCount++
	}
	if hasDigit {
		checkCount++
	}
	if hasSpecial {
		checkCount++
	}
	
	return checkCount >= 3
}

// IsValidURL validates URL format
func IsValidURL(url string) bool {
	url = strings.TrimSpace(url)
	if len(url) == 0 {
		return false
	}
	
	// Simple URL validation
	urlPattern := `^https?://[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}(/.*)?$`
	matched, _ := regexp.MatchString(urlPattern, url)
	return matched
}

// SanitizeString removes dangerous characters and trims whitespace
func SanitizeString(input string) string {
	// Remove null bytes and trim whitespace
	sanitized := strings.ReplaceAll(input, "\x00", "")
	return strings.TrimSpace(sanitized)
}

// IsValidObjectID checks if a string is a valid MongoDB ObjectID
func IsValidObjectID(id string) bool {
	if len(id) != 24 {
		return false
	}
	
	for _, char := range id {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
			return false
		}
	}
	
	return true
}

// ValidateRequiredString validates that a string is not empty after trimming
func ValidateRequiredString(value, fieldName string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is required", fieldName)
	}
	return nil
}

// ValidateStringLength validates string length constraints
func ValidateStringLength(value, fieldName string, min, max int) error {
	length := len(strings.TrimSpace(value))
	
	if length < min {
		return fmt.Errorf("%s must be at least %d characters long", fieldName, min)
	}
	
	if max > 0 && length > max {
		return fmt.Errorf("%s cannot exceed %d characters", fieldName, max)
	}
	
	return nil
}

// ValidateNumericRange validates that a number is within a specified range
func ValidateNumericRange(value int, fieldName string, min, max int) error {
	if value < min {
		return fmt.Errorf("%s must be at least %d", fieldName, min)
	}
	
	if max > 0 && value > max {
		return fmt.Errorf("%s cannot exceed %d", fieldName, max)
	}
	
	return nil
}