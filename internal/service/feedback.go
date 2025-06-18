package service

import (
	"context"
	"fmt"
	"time"

	"github.com/Ravwvil/feedback/internal/models"
	"github.com/Ravwvil/feedback/internal/repository"
	"github.com/Ravwvil/feedback/internal/storage"
)

type FeedbackService struct {
	repo        *repository.FeedbackRepository
	minioClient *storage.MinIOClient
}

type CreateFeedbackParams struct {
	UserID      int64
	LabID       int64
	Title       string
	Content     string
	ContentHash string
}

type UpdateFeedbackParams struct {
	ID          string
	Title       string
	Content     string
	ContentHash string
}

type ListUserFeedbacksParams struct {
	UserID int64
	LabID  int64
	Page   int
	Limit  int
}

func NewFeedbackService(repo *repository.FeedbackRepository, minioClient *storage.MinIOClient) *FeedbackService {
	return &FeedbackService{
		repo:        repo,
		minioClient: minioClient,
	}
}

func (s *FeedbackService) CreateFeedback(ctx context.Context, params *CreateFeedbackParams) (*models.FeedbackFile, error) {
	feedback := &models.FeedbackFile{
		UserID:      params.UserID,
		LabID:       params.LabID,
		Title:       params.Title,
		Content:     params.Content,
		ContentHash: params.ContentHash,
	}

	// Save metadata to database
	err := s.repo.Create(ctx, feedback)
	if err != nil {
		return nil, fmt.Errorf("failed to create feedback in database: %w", err)
	}

	// Save content to MinIO
	err = s.minioClient.UploadFile(ctx, feedback.ID, "content.md", "text/markdown", []byte(params.Content))
	if err != nil {
		// Rollback database record if MinIO upload fails
		s.repo.Delete(ctx, feedback.ID)
		return nil, fmt.Errorf("failed to upload content to storage: %w", err)
	}

	return feedback, nil
}

func (s *FeedbackService) GetFeedback(ctx context.Context, id string) (*models.FeedbackFile, error) {
	// Get metadata from database
	feedback, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get feedback metadata: %w", err)
	}

	// Get content from MinIO
	content, err := s.minioClient.DownloadFile(ctx, id, "content.md")
	if err != nil {
		return nil, fmt.Errorf("failed to download content from storage: %w", err)
	}

	feedback.Content = string(content)
	return feedback, nil
}

func (s *FeedbackService) UpdateFeedback(ctx context.Context, params *UpdateFeedbackParams) (*models.FeedbackFile, error) {
	// Get existing feedback
	feedback, err := s.repo.GetByID(ctx, params.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing feedback: %w", err)
	}

	// Update fields if provided
	if params.Title != "" {
		feedback.Title = params.Title
	}
	if params.Content != "" {
		feedback.Content = params.Content
		feedback.ContentHash = params.ContentHash
	}

	// Update database
	err = s.repo.Update(ctx, feedback)
	if err != nil {
		return nil, fmt.Errorf("failed to update feedback in database: %w", err)
	}

	// Update content in MinIO if provided
	if params.Content != "" {
		err = s.minioClient.UploadFile(ctx, feedback.ID, "content.md", "text/markdown", []byte(params.Content))
		if err != nil {
			return nil, fmt.Errorf("failed to update content in storage: %w", err)
		}
	}

	return feedback, nil
}

func (s *FeedbackService) DeleteFeedback(ctx context.Context, id string) error {
	// Delete from MinIO first (folder and all assets)
	err := s.minioClient.DeleteFolder(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete content from storage: %w", err)
	}

	// Delete from database
	err = s.repo.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete feedback from database: %w", err)
	}

	return nil
}

func (s *FeedbackService) ListUserFeedbacks(ctx context.Context, params *ListUserFeedbacksParams) ([]*models.FeedbackFile, int, error) {
	// Set default pagination
	if params.Limit <= 0 {
		params.Limit = 20
	}
	if params.Page <= 0 {
		params.Page = 1
	}

	offset := (params.Page - 1) * params.Limit

	return s.repo.ListByUserID(ctx, params.UserID, params.LabID, offset, params.Limit)
}

func (s *FeedbackService) UploadAsset(ctx context.Context, feedbackID, filename, contentType string, data []byte) (int64, error) {
	assetPath := fmt.Sprintf("assets/%s", filename)
	
	err := s.minioClient.UploadFile(ctx, feedbackID, assetPath, contentType, data)
	if err != nil {
		return 0, fmt.Errorf("failed to upload asset: %w", err)
	}

	return int64(len(data)), nil
}

func (s *FeedbackService) DownloadAsset(ctx context.Context, feedbackID, filename string) (*models.AssetInfo, []byte, error) {
	assetPath := fmt.Sprintf("assets/%s", filename)
	
	data, err := s.minioClient.DownloadFile(ctx, feedbackID, assetPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to download asset: %w", err)
	}

	// Get asset info
	info, err := s.minioClient.GetFileInfo(ctx, feedbackID, assetPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get asset info: %w", err)
	}

	assetInfo := &models.AssetInfo{
		Filename:    filename,
		Size:        info.Size,
		ContentType: info.ContentType,
		UploadedAt:  info.LastModified,
	}

	return assetInfo, data, nil
}

func (s *FeedbackService) ListAssets(ctx context.Context, feedbackID string) ([]*models.AssetInfo, error) {
	files, err := s.minioClient.ListFiles(ctx, feedbackID, "assets/")
	if err != nil {
		return nil, fmt.Errorf("failed to list assets: %w", err)
	}

	assets := make([]*models.AssetInfo, len(files))
	for i, file := range files {
		assets[i] = &models.AssetInfo{
			Filename:    file.Filename,
			Size:        file.Size,
			ContentType: file.ContentType,
			UploadedAt:  file.LastModified,
		}
	}

	return assets, nil
}
