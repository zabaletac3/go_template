// internal/models/dto.go
package models

import (
	"encoding/json"
	"strings"
	"time"
)

// CreateUserRequest represents the request payload for creating a user
type CreateUserRequest struct {
	Username  string `json:"username" validate:"required,min=3,max=30" example:"johndoe"`
	Email     string `json:"email" validate:"required,email,max=255" example:"john@example.com"`
	Password  string `json:"password" validate:"required,min=8,max=128" example:"SecurePass123"`
	FirstName string `json:"first_name,omitempty" validate:"max=50" example:"John"`
	LastName  string `json:"last_name,omitempty" validate:"max=50" example:"Doe"`
}

// UpdateUserRequest represents the request payload for updating a user
type UpdateUserRequest struct {
	Username  *string `json:"username,omitempty" validate:"omitempty,min=3,max=30" example:"janedoe"`
	Email     *string `json:"email,omitempty" validate:"omitempty,email,max=255" example:"jane@example.com"`
	FirstName *string `json:"first_name,omitempty" validate:"omitempty,max=50" example:"Jane"`
	LastName  *string `json:"last_name,omitempty" validate:"omitempty,max=50" example:"Smith"`
	Bio       *string `json:"bio,omitempty" validate:"omitempty,max=500" example:"Software developer and coffee enthusiast"`
	Location  *string `json:"location,omitempty" validate:"omitempty,max=100" example:"San Francisco, CA"`
	Website   *string `json:"website,omitempty" validate:"omitempty,url,max=255" example:"https://johndoe.dev"`
}

// ChangePasswordRequest represents the request payload for changing password
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required" example:"OldPassword123"`
	NewPassword     string `json:"new_password" validate:"required,min=8,max=128" example:"NewSecurePassword456"`
	ConfirmPassword string `json:"confirm_password" validate:"required" example:"NewSecurePassword456"`
}

// LoginRequest represents the request payload for user login
type LoginRequest struct {
	Username string `json:"username" validate:"required" example:"johndoe"`
	Password string `json:"password" validate:"required" example:"SecurePass123"`
}

// UserResponse represents the response payload for user data
type UserResponse struct {
	ID              string                 `json:"id"`
	Username        string                 `json:"username"`
	Email           string                 `json:"email"`
	FirstName       string                 `json:"first_name"`
	LastName        string                 `json:"last_name"`
	FullName        string                 `json:"full_name"`
	Avatar          string                 `json:"avatar"`
	Bio             string                 `json:"bio"`
	Location        string                 `json:"location"`
	Website         string                 `json:"website"`
	DateOfBirth     *time.Time             `json:"date_of_birth"`
	IsActive        bool                   `json:"is_active"`
	IsVerified      bool                   `json:"is_verified"`
	Roles           []string               `json:"roles"`
	LastLoginAt     *time.Time             `json:"last_login_at"`
	EmailVerifiedAt *time.Time             `json:"email_verified_at"`
	LoginCount      int                    `json:"login_count"`
	Preferences     map[string]interface{} `json:"preferences"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// UserListResponse represents the response for user list queries
type UserListResponse struct {
	Users []UserResponse `json:"users"`
	Total int            `json:"total"`
	Page  int            `json:"page"`
	Limit int            `json:"limit"`
}

// UserProfileResponse represents a public user profile (limited information)
type UserProfileResponse struct {
	ID          string     `json:"id"`
	Username    string     `json:"username"`
	FullName    string     `json:"full_name"`
	Avatar      string     `json:"avatar"`
	Bio         string     `json:"bio"`
	Location    string     `json:"location"`
	Website     string     `json:"website"`
	IsVerified  bool       `json:"is_verified"`
	CreatedAt   time.Time  `json:"created_at"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
}

// LoginResponse represents the response payload for successful login
type LoginResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	TokenType    string       `json:"token_type"`
	ExpiresIn    int          `json:"expires_in"`
	User         UserResponse `json:"user"`
}

// UsersQueryParams represents query parameters for user listing
type UsersQueryParams struct {
	Page     int    `json:"page" validate:"min=1"`
	Limit    int    `json:"limit" validate:"min=1,max=100"`
	Search   string `json:"search,omitempty"`
	Role     string `json:"role,omitempty"`
	IsActive *bool  `json:"is_active,omitempty"`
	SortBy   string `json:"sort_by,omitempty"`
	SortDir  string `json:"sort_dir,omitempty"`
}

// Conversion methods

// ToUserResponse converts a User model to UserResponse DTO
func (u *User) ToUserResponse() UserResponse {
	return UserResponse{
		ID:              u.GetIDString(),
		Username:        u.Username,
		Email:           u.Email,
		FirstName:       u.FirstName,
		LastName:        u.LastName,
		FullName:        u.GetFullName(),
		Avatar:          u.Avatar,
		Bio:             u.Bio,
		Location:        u.Location,
		Website:         u.Website,
		DateOfBirth:     u.DateOfBirth,
		IsActive:        u.IsActive,
		IsVerified:      u.IsVerified,
		Roles:           u.Roles,
		LastLoginAt:     u.LastLoginAt,
		EmailVerifiedAt: u.EmailVerifiedAt,
		LoginCount:      u.LoginCount,
		Preferences:     u.Preferences,
		CreatedAt:       u.CreatedAt,
		UpdatedAt:       u.UpdatedAt,
	}
}

// ToUserProfileResponse converts a User model to UserProfileResponse DTO (public profile)
func (u *User) ToUserProfileResponse() UserProfileResponse {
	profile := UserProfileResponse{
		ID:         u.GetIDString(),
		Username:   u.Username,
		FullName:   u.GetFullName(),
		Avatar:     u.Avatar,
		Bio:        u.Bio,
		Location:   u.Location,
		Website:    u.Website,
		IsVerified: u.IsVerified,
		CreatedAt:  u.CreatedAt,
	}
	
	// Only include last login for active users (privacy)
	if u.IsActive {
		profile.LastLoginAt = u.LastLoginAt
	}
	
	return profile
}

// ToMap converts UpdateUserRequest to a map for partial updates
func (r *UpdateUserRequest) ToMap() map[string]interface{} {
	updates := make(map[string]interface{})
	
	if r.Username != nil {
		updates["username"] = strings.TrimSpace(*r.Username)
	}
	if r.Email != nil {
		updates["email"] = strings.TrimSpace(*r.Email)
	}
	if r.FirstName != nil {
		updates["first_name"] = strings.TrimSpace(*r.FirstName)
	}
	if r.LastName != nil {
		updates["last_name"] = strings.TrimSpace(*r.LastName)
	}
	if r.Bio != nil {
		updates["bio"] = strings.TrimSpace(*r.Bio)
	}
	if r.Location != nil {
		updates["location"] = strings.TrimSpace(*r.Location)
	}
	if r.Website != nil {
		updates["website"] = strings.TrimSpace(*r.Website)
	}
	
	return updates
}

// Validate validates the CreateUserRequest
func (r *CreateUserRequest) Validate() []string {
	var errors []string
	
	// Trim spaces
	r.Username = strings.TrimSpace(r.Username)
	r.Email = strings.TrimSpace(r.Email)
	r.FirstName = strings.TrimSpace(r.FirstName)
	r.LastName = strings.TrimSpace(r.LastName)
	
	// Validate username
	if err := ValidateUsername(r.Username); err != nil {
		errors = append(errors, err.Error())
	}
	
	// Validate email
	if err := ValidateEmail(r.Email); err != nil {
		errors = append(errors, err.Error())
	}
	
	// Validate password
	if err := ValidatePassword(r.Password); err != nil {
		errors = append(errors, err.Error())
	}
	
	// Validate optional fields
	if r.FirstName != "" && len(r.FirstName) > 50 {
		errors = append(errors, "first name cannot exceed 50 characters")
	}
	
	if r.LastName != "" && len(r.LastName) > 50 {
		errors = append(errors, "last name cannot exceed 50 characters")
	}
	
	return errors
}

// Validate validates the UpdateUserRequest
func (r *UpdateUserRequest) Validate() []string {
	var errors []string
	
	if r.Username != nil {
		*r.Username = strings.TrimSpace(*r.Username)
		if err := ValidateUsername(*r.Username); err != nil {
			errors = append(errors, err.Error())
		}
	}
	
	if r.Email != nil {
		*r.Email = strings.TrimSpace(*r.Email)
		if err := ValidateEmail(*r.Email); err != nil {
			errors = append(errors, err.Error())
		}
	}
	
	if r.FirstName != nil {
		*r.FirstName = strings.TrimSpace(*r.FirstName)
		if len(*r.FirstName) > 50 {
			errors = append(errors, "first name cannot exceed 50 characters")
		}
	}
	
	if r.LastName != nil {
		*r.LastName = strings.TrimSpace(*r.LastName)
		if len(*r.LastName) > 50 {
			errors = append(errors, "last name cannot exceed 50 characters")
		}
	}
	
	if r.Bio != nil {
		*r.Bio = strings.TrimSpace(*r.Bio)
		if len(*r.Bio) > 500 {
			errors = append(errors, "bio cannot exceed 500 characters")
		}
	}
	
	if r.Location != nil {
		*r.Location = strings.TrimSpace(*r.Location)
		if len(*r.Location) > 100 {
			errors = append(errors, "location cannot exceed 100 characters")
		}
	}
	
	if r.Website != nil {
		*r.Website = strings.TrimSpace(*r.Website)
		if *r.Website != "" && !isValidURL(*r.Website) {
			errors = append(errors, "invalid website URL format")
		}
	}
	
	return errors
}

// Validate validates the ChangePasswordRequest
func (r *ChangePasswordRequest) Validate() []string {
	var errors []string
	
	if r.CurrentPassword == "" {
		errors = append(errors, "current password is required")
	}
	
	if err := ValidatePassword(r.NewPassword); err != nil {
		errors = append(errors, err.Error())
	}
	
	if r.NewPassword != r.ConfirmPassword {
		errors = append(errors, "new password and confirm password do not match")
	}
	
	if r.CurrentPassword == r.NewPassword {
		errors = append(errors, "new password must be different from current password")
	}
	
	return errors
}

// Validate validates the LoginRequest
func (r *LoginRequest) Validate() []string {
	var errors []string
	
	r.Username = strings.TrimSpace(r.Username)
	
	if r.Username == "" {
		errors = append(errors, "username or email is required")
	}
	
	if r.Password == "" {
		errors = append(errors, "password is required")
	}
	
	return errors
}

// Default values for query parameters
func (q *UsersQueryParams) SetDefaults() {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.Limit < 1 || q.Limit > 100 {
		q.Limit = 20
	}
	if q.SortBy == "" {
		q.SortBy = "created_at"
	}
	if q.SortDir == "" {
		q.SortDir = "desc"
	}
}

// JSON marshaling customization for sensitive fields

// MarshalJSON customizes JSON output for CreateUserRequest (excludes password in logs)
func (r CreateUserRequest) MarshalJSON() ([]byte, error) {
	type SafeCreateUserRequest struct {
		Username  string `json:"username"`
		Email     string `json:"email"`
		FirstName string `json:"first_name,omitempty"`
		LastName  string `json:"last_name,omitempty"`
		Password  string `json:"password,omitempty"`
	}
	
	safe := SafeCreateUserRequest{
		Username:  r.Username,
		Email:     r.Email,
		FirstName: r.FirstName,
		LastName:  r.LastName,
		Password:  "[REDACTED]",
	}
	
	return json.Marshal(safe)
}