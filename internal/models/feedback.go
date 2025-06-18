package models

import (
	"time"
)

// FeedbackFile represents a feedback file stored in the system
type FeedbackFile struct {
	ID          string    `json:"id" db:"id"`
	UserID      int64     `json:"user_id" db:"user_id"`
	LabID       int64     `json:"lab_id" db:"lab_id"`
	Title       string    `json:"title" db:"title"`
	Content     string    `json:"content"`                    // Markdown content (stored in MinIO)
	ContentHash string    `json:"content_hash" db:"content_hash"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// AssetInfo represents information about an uploaded asset
type AssetInfo struct {
	Filename    string    `json:"filename"`
	Size        int64     `json:"size"`
	ContentType string    `json:"content_type"`
	UploadedAt  time.Time `json:"uploaded_at"`
}
