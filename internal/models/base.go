// internal/models/base.go
package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// BaseModel contains common fields for all models
type BaseModel struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	CreatedAt time.Time         `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time         `json:"updated_at" bson:"updated_at"`
	DeletedAt *time.Time        `json:"deleted_at,omitempty" bson:"deleted_at,omitempty"`
}

// NewBaseModel creates a new base model with current timestamps
func NewBaseModel() BaseModel {
	now := time.Now().UTC()
	return BaseModel{
		ID:        primitive.NewObjectID(),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// UpdateTimestamp updates the UpdatedAt field to current time
func (b *BaseModel) UpdateTimestamp() {
	b.UpdatedAt = time.Now().UTC()
}

// SoftDelete marks the model as deleted by setting DeletedAt
func (b *BaseModel) SoftDelete() {
	now := time.Now().UTC()
	b.DeletedAt = &now
	b.UpdatedAt = now
}

// IsDeleted returns true if the model is soft deleted
func (b *BaseModel) IsDeleted() bool {
	return b.DeletedAt != nil
}

// GetIDString returns the ID as a string
func (b *BaseModel) GetIDString() string {
	return b.ID.Hex()
}

// IsValidObjectID checks if a string is a valid MongoDB ObjectID
func IsValidObjectID(id string) bool {
	_, err := primitive.ObjectIDFromHex(id)
	return err == nil
}

// ObjectIDFromString converts a string to ObjectID with error handling
func ObjectIDFromString(id string) (primitive.ObjectID, error) {
	return primitive.ObjectIDFromHex(id)
}