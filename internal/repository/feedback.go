package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Ravwvil/feedback/internal/models"
	"github.com/google/uuid"
)

type FeedbackRepository struct {
	db *sql.DB
}

func NewFeedbackRepository(db *sql.DB) *FeedbackRepository {
	return &FeedbackRepository{
		db: db,
	}
}

func (r *FeedbackRepository) Create(ctx context.Context, feedback *models.FeedbackFile) error {
	feedback.ID = uuid.New().String()
	
	query := `
		INSERT INTO feedback_files (id, user_id, lab_id, title, content_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		RETURNING created_at, updated_at`
	
	err := r.db.QueryRowContext(ctx, query,
		feedback.ID,
		feedback.UserID,
		feedback.LabID,
		feedback.Title,
		feedback.ContentHash,
	).Scan(&feedback.CreatedAt, &feedback.UpdatedAt)
	
	return err
}

func (r *FeedbackRepository) GetByID(ctx context.Context, id string) (*models.FeedbackFile, error) {
	feedback := &models.FeedbackFile{}
	
	query := `
		SELECT id, user_id, lab_id, title, content_hash, created_at, updated_at
		FROM feedback_files
		WHERE id = $1`
	
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&feedback.ID,
		&feedback.UserID,
		&feedback.LabID,
		&feedback.Title,
		&feedback.ContentHash,
		&feedback.CreatedAt,
		&feedback.UpdatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	return feedback, nil
}

func (r *FeedbackRepository) Update(ctx context.Context, feedback *models.FeedbackFile) error {
	query := `
		UPDATE feedback_files
		SET title = $2, content_hash = $3, updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at`
	
	err := r.db.QueryRowContext(ctx, query,
		feedback.ID,
		feedback.Title,
		feedback.ContentHash,
	).Scan(&feedback.UpdatedAt)
	
	return err
}

func (r *FeedbackRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM feedback_files WHERE id = $1`
	
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("feedback with id %s not found", id)
	}
	
	return nil
}

func (r *FeedbackRepository) ListByUserID(ctx context.Context, userID int64, labID int64, offset, limit int) ([]*models.FeedbackFile, int, error) {
	var feedbacks []*models.FeedbackFile
	var args []interface{}
	
	// Build query with optional lab_id filter
	whereClause := "WHERE user_id = $1"
	args = append(args, userID)
	
	if labID > 0 {
		whereClause += " AND lab_id = $2"
		args = append(args, labID)
	}
	
	// Count total records
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM feedback_files %s", whereClause)
	var totalCount int
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, err
	}
	
	// Get paginated results
	query := fmt.Sprintf(`
		SELECT id, user_id, lab_id, title, content_hash, created_at, updated_at
		FROM feedback_files
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`, whereClause, len(args)+1, len(args)+2)
	
	args = append(args, limit, offset)
	
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	
	for rows.Next() {
		feedback := &models.FeedbackFile{}
		err := rows.Scan(
			&feedback.ID,
			&feedback.UserID,
			&feedback.LabID,
			&feedback.Title,
			&feedback.ContentHash,
			&feedback.CreatedAt,
			&feedback.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		feedbacks = append(feedbacks, feedback)
	}
	
	return feedbacks, totalCount, rows.Err()
}
