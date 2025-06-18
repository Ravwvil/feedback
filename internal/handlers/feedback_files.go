package handlers

import (
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/Ravwvil/feedback/internal/middleware"
	"github.com/Ravwvil/feedback/internal/models"
	"github.com/Ravwvil/feedback/internal/service"
)

type FeedbackFileHandler struct {
	service *service.FeedbackService
}

func NewFeedbackFileHandler(service *service.FeedbackService) *FeedbackFileHandler {
	return &FeedbackFileHandler{service: service}
}

// UploadFeedbackFile handles POST /feedback/files/upload
func (h *FeedbackFileHandler) UploadFeedbackFile(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not provided by API Gateway"})
		return
	}

	var req models.FeedbackFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	feedback, err := h.service.UploadFeedbackFile(c.Request.Context(), userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload feedback file", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":  "Feedback file uploaded successfully",
		"feedback": feedback,
	})
}

// GetFeedbackFile handles GET /feedback/files/{feedbackId}
func (h *FeedbackFileHandler) GetFeedbackFile(c *gin.Context) {
	feedbackID := c.Param("feedbackId")
	if feedbackID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Feedback ID is required"})
		return
	}

	feedback, err := h.service.GetFeedbackFile(c.Request.Context(), feedbackID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Feedback file not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve feedback file", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"feedback": feedback})
}

// UpdateFeedbackFile handles PUT /feedback/files/{feedbackId}
func (h *FeedbackFileHandler) UpdateFeedbackFile(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not provided by API Gateway"})
		return
	}

	feedbackID := c.Param("feedbackId")
	if feedbackID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Feedback ID is required"})
		return
	}

	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	err = h.service.UpdateFeedbackFile(c.Request.Context(), feedbackID, userID, req.Content)
	if err != nil {
		if strings.Contains(err.Error(), "unauthorized") {
			c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized: You don't own this feedback"})
			return
		}
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Feedback file not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update feedback file", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Feedback file updated successfully"})
}

// DeleteFeedbackFile handles DELETE /feedback/files/{feedbackId}
func (h *FeedbackFileHandler) DeleteFeedbackFile(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not provided by API Gateway"})
		return
	}

	feedbackID := c.Param("feedbackId")
	if feedbackID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Feedback ID is required"})
		return
	}

	err = h.service.DeleteFeedbackFile(c.Request.Context(), feedbackID, userID)
	if err != nil {
		if strings.Contains(err.Error(), "unauthorized") {
			c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized: You don't own this feedback"})
			return
		}
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Feedback file not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete feedback file", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Feedback file deleted successfully"})
}

// UploadAsset handles POST /feedback/files/{feedbackId}/assets
func (h *FeedbackFileHandler) UploadAsset(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not provided by API Gateway"})
		return
	}

	feedbackID := c.Param("feedbackId")
	if feedbackID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Feedback ID is required"})
		return
	}

	// Parse multipart form
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse multipart form", "details": err.Error()})
		return
	}

	files := form.File["asset"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No asset file provided"})
		return
	}

	var uploadedFiles []string
	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to open uploaded file", "details": err.Error()})
			return
		}
		defer file.Close()

		data, err := io.ReadAll(file)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file content", "details": err.Error()})
			return
		}

		contentType := fileHeader.Header.Get("Content-Type")
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		err = h.service.UploadAsset(c.Request.Context(), feedbackID, userID, fileHeader.Filename, data, contentType)
		if err != nil {
			if strings.Contains(err.Error(), "unauthorized") {
				c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized: You don't own this feedback"})
				return
			}
			if strings.Contains(err.Error(), "not found") {
				c.JSON(http.StatusNotFound, gin.H{"error": "Feedback file not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload asset", "details": err.Error()})
			return
		}

		uploadedFiles = append(uploadedFiles, fileHeader.Filename)
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Assets uploaded successfully",
		"files":   uploadedFiles,
	})
}

// GetAsset handles GET /feedback/files/{feedbackId}/assets/{filename}
func (h *FeedbackFileHandler) GetAsset(c *gin.Context) {
	feedbackID := c.Param("feedbackId")
	filename := c.Param("filename")

	if feedbackID == "" || filename == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Feedback ID and filename are required"})
		return
	}

	data, contentType, err := h.service.GetAsset(c.Request.Context(), feedbackID, filename)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Asset not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve asset", "details": err.Error()})
		return
	}

	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", "inline; filename=\""+filepath.Base(filename)+"\"")
	c.Data(http.StatusOK, contentType, data)
}

// DeleteAsset handles DELETE /feedback/files/{feedbackId}/assets/{filename}
func (h *FeedbackFileHandler) DeleteAsset(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not provided by API Gateway"})
		return
	}

	feedbackID := c.Param("feedbackId")
	filename := c.Param("filename")

	if feedbackID == "" || filename == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Feedback ID and filename are required"})
		return
	}

	err = h.service.DeleteAsset(c.Request.Context(), feedbackID, filename, userID)
	if err != nil {
		if strings.Contains(err.Error(), "unauthorized") {
			c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized: You don't own this feedback"})
			return
		}
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Asset or feedback not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete asset", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Asset deleted successfully"})
}

// ListUserFeedbacks handles GET /feedback/files
func (h *FeedbackFileHandler) ListUserFeedbacks(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not provided by API Gateway"})
		return
	}

	page, limit := middleware.GetPaginationParams(c)

	feedbacks, err := h.service.ListUserFeedbacks(userID, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve feedbacks", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"feedbacks": feedbacks,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
		},
	})
}
