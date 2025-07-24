// internal/models/user.go
package models

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// User represents a user in the system
type User struct {
	BaseModel `bson:",inline"`
	
	// Basic Information
	Username    string `json:"username" bson:"username"`
	Email       string `json:"email" bson:"email"`
	FirstName   string `json:"first_name" bson:"first_name"`
	LastName    string `json:"last_name" bson:"last_name"`
	
	// Authentication
	Password    string `json:"-" bson:"password"` // Never send password in JSON
	Salt        string `json:"-" bson:"salt"`     // Password salt
	
	// Profile Information
	Avatar      string    `json:"avatar" bson:"avatar"`
	Bio         string    `json:"bio" bson:"bio"`
	Location    string    `json:"location" bson:"location"`
	Website     string    `json:"website" bson:"website"`
	DateOfBirth *time.Time `json:"date_of_birth" bson:"date_of_birth"`
	
	// Status and Permissions
	IsActive    bool     `json:"is_active" bson:"is_active"`
	IsVerified  bool     `json:"is_verified" bson:"is_verified"`
	Roles       []string `json:"roles" bson:"roles"`
	
	// Timestamps for specific actions
	LastLoginAt    *time.Time `json:"last_login_at" bson:"last_login_at"`
	EmailVerifiedAt *time.Time `json:"email_verified_at" bson:"email_verified_at"`
	
	// Metadata
	LoginCount     int               `json:"login_count" bson:"login_count"`
	FailedLogins   int               `json:"-" bson:"failed_logins"`
	LastFailedAt   *time.Time        `json:"-" bson:"last_failed_at"`
	Preferences    map[string]interface{} `json:"preferences" bson:"preferences"`
}

// UserRole constants
const (
	RoleUser  = "user"
	RoleAdmin = "admin"
	RoleMod   = "moderator"
)

// NewUser creates a new user with default values
func NewUser(username, email, password string) (*User, error) {
	// Validate input
	if err := ValidateUsername(username); err != nil {
		return nil, err
	}
	
	if err := ValidateEmail(email); err != nil {
		return nil, err
	}
	
	if err := ValidatePassword(password); err != nil {
		return nil, err
	}
	
	// Generate salt and hash password
	salt, err := generateSalt()
	if err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}
	
	hashedPassword := hashPassword(password, salt)
	
	user := &User{
		BaseModel: NewBaseModel(),
		Username:  strings.ToLower(strings.TrimSpace(username)),
		Email:     strings.ToLower(strings.TrimSpace(email)),
		Password:  hashedPassword,
		Salt:      salt,
		IsActive:  true,
		IsVerified: false,
		Roles:     []string{RoleUser}, // Default role
		LoginCount: 0,
		FailedLogins: 0,
		Preferences: make(map[string]interface{}),
	}
	
	return user, nil
}

// UpdateUser updates user fields and timestamp
func (u *User) UpdateUser(updates map[string]interface{}) error {
	u.UpdateTimestamp()
	
	// Update allowed fields
	if username, ok := updates["username"].(string); ok {
		if err := ValidateUsername(username); err != nil {
			return err
		}
		u.Username = strings.ToLower(strings.TrimSpace(username))
	}
	
	if email, ok := updates["email"].(string); ok {
		if err := ValidateEmail(email); err != nil {
			return err
		}
		// If email changed, mark as unverified
		if u.Email != strings.ToLower(strings.TrimSpace(email)) {
			u.IsVerified = false
			u.EmailVerifiedAt = nil
		}
		u.Email = strings.ToLower(strings.TrimSpace(email))
	}
	
	if firstName, ok := updates["first_name"].(string); ok {
		u.FirstName = strings.TrimSpace(firstName)
	}
	
	if lastName, ok := updates["last_name"].(string); ok {
		u.LastName = strings.TrimSpace(lastName)
	}
	
	if bio, ok := updates["bio"].(string); ok {
		if len(bio) > 500 {
			return errors.New("bio cannot exceed 500 characters")
		}
		u.Bio = strings.TrimSpace(bio)
	}
	
	if location, ok := updates["location"].(string); ok {
		u.Location = strings.TrimSpace(location)
	}
	
	if website, ok := updates["website"].(string); ok {
		if website != "" && !isValidURL(website) {
			return errors.New("invalid website URL format")
		}
		u.Website = strings.TrimSpace(website)
	}
	
	return nil
}

// SetPassword updates the user's password with proper hashing
func (u *User) SetPassword(newPassword string) error {
	if err := ValidatePassword(newPassword); err != nil {
		return err
	}
	
	// Generate new salt
	salt, err := generateSalt()
	if err != nil {
		return fmt.Errorf("failed to generate salt: %w", err)
	}
	
	u.Salt = salt
	u.Password = hashPassword(newPassword, salt)
	u.UpdateTimestamp()
	
	return nil
}

// CheckPassword verifies if the provided password matches the user's password
func (u *User) CheckPassword(password string) bool {
	hashedInput := hashPassword(password, u.Salt)
	return u.Password == hashedInput
}

// RecordLogin updates login-related fields
func (u *User) RecordLogin() {
	now := time.Now().UTC()
	u.LastLoginAt = &now
	u.LoginCount++
	u.FailedLogins = 0 // Reset failed login attempts
	u.UpdateTimestamp()
}

// RecordFailedLogin increments failed login counter
func (u *User) RecordFailedLogin() {
	now := time.Now().UTC()
	u.FailedLogins++
	u.LastFailedAt = &now
	u.UpdateTimestamp()
}

// IsLocked returns true if user account is locked due to failed logins
func (u *User) IsLocked() bool {
	const maxFailedLogins = 5
	const lockoutDuration = 30 * time.Minute
	
	if u.FailedLogins < maxFailedLogins {
		return false
	}
	
	if u.LastFailedAt == nil {
		return false
	}
	
	return time.Since(*u.LastFailedAt) < lockoutDuration
}

// VerifyEmail marks the user's email as verified
func (u *User) VerifyEmail() {
	now := time.Now().UTC()
	u.IsVerified = true
	u.EmailVerifiedAt = &now
	u.UpdateTimestamp()
}

// GetFullName returns the user's full name
func (u *User) GetFullName() string {
	fullName := strings.TrimSpace(u.FirstName + " " + u.LastName)
	if fullName == "" {
		return u.Username
	}
	return fullName
}

// HasRole checks if user has a specific role
func (u *User) HasRole(role string) bool {
	for _, r := range u.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// AddRole adds a role to the user if not already present
func (u *User) AddRole(role string) {
	if !u.HasRole(role) {
		u.Roles = append(u.Roles, role)
		u.UpdateTimestamp()
	}
}

// RemoveRole removes a role from the user
func (u *User) RemoveRole(role string) {
	for i, r := range u.Roles {
		if r == role {
			u.Roles = append(u.Roles[:i], u.Roles[i+1:]...)
			u.UpdateTimestamp()
			break
		}
	}
}

// IsAdmin returns true if user has admin role
func (u *User) IsAdmin() bool {
	return u.HasRole(RoleAdmin)
}

// Validation functions

// ValidateUsername validates username format and length
func ValidateUsername(username string) error {
	username = strings.TrimSpace(username)
	
	if len(username) < 3 {
		return errors.New("username must be at least 3 characters long")
	}
	
	if len(username) > 30 {
		return errors.New("username cannot exceed 30 characters")
	}
	
	// Username can only contain letters, numbers, and underscores
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_]+$`, username)
	if !matched {
		return errors.New("username can only contain letters, numbers, and underscores")
	}
	
	return nil
}

// ValidateEmail validates email format
func ValidateEmail(email string) error {
	email = strings.TrimSpace(email)
	
	if email == "" {
		return errors.New("email is required")
	}
	
	// Basic email regex (more comprehensive than simple check)
	emailRegex := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	matched, _ := regexp.MatchString(emailRegex, email)
	if !matched {
		return errors.New("invalid email format")
	}
	
	if len(email) > 255 {
		return errors.New("email cannot exceed 255 characters")
	}
	
	return nil
}

// ValidatePassword validates password strength
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters long")
	}
	
	if len(password) > 128 {
		return errors.New("password cannot exceed 128 characters")
	}
	
	// Check for at least one uppercase, one lowercase, and one digit
	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	hasDigit := regexp.MustCompile(`\d`).MatchString(password)
	
	if !hasUpper || !hasLower || !hasDigit {
		return errors.New("password must contain at least one uppercase letter, one lowercase letter, and one digit")
	}
	
	return nil
}

// Helper functions

// generateSalt generates a random salt for password hashing
func generateSalt() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// hashPassword creates a SHA-256 hash of password + salt
func hashPassword(password, salt string) string {
	hash := sha256.Sum256([]byte(password + salt))
	return hex.EncodeToString(hash[:])
}

// isValidURL checks if a string is a valid URL
func isValidURL(url string) bool {
	urlRegex := `^https?://[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}(/.*)?$`
	matched, _ := regexp.MatchString(urlRegex, url)
	return matched
}