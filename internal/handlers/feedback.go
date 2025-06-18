package handlers

import (
	"net/http"
	"strconv"
	
	"github.com/gin-gonic/gin"
	"github.com/Ravwvil/feedback/internal/middleware"
	"github.com/Ravwvil/feedback/internal/models"
	"github.com/Ravwvil/feedback/internal/service"
)

type FeedbackHandler struct {
	service *service.FeedbackService
}

func NewFeedbackHandler(service *service.FeedbackService) *FeedbackHandler {
	return &FeedbackHandler{service: service}
}

// SubmitReview handles POST /feedback/submit
func (h *FeedbackHandler) SubmitReview(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not provided by API Gateway"})
		return
	}
	
	var req models.ReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}
	
	review, err := h.service.SubmitReview(userID, &req)
	if err != nil {
		if err.Error() == "user has already reviewed this submission" {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit review", "details": err.Error()})
		return
	}
	
	c.JSON(http.StatusCreated, gin.H{
		"message": "Review submitted successfully",
		"review":  review,
	})
}

// GetLabReviews handles GET /feedback/{labId}
func (h *FeedbackHandler) GetLabReviews(c *gin.Context) {
	labIDStr := c.Param("labId")
	labID, err := strconv.ParseInt(labIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid lab ID"})
		return
	}
	
	page, limit := middleware.GetPaginationParams(c)
	
	reviews, err := h.service.GetLabReviews(labID, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get reviews", "details": err.Error()})
		return
	}
	
	// Get summary statistics
	summary, err := h.service.GetLabReviewSummary(labID)
	if err != nil {
		// Don't fail the request if summary fails
		summary = &models.LabReviewSummary{LabID: labID}
	}
	
	c.JSON(http.StatusOK, gin.H{
		"reviews":    reviews,
		"summary":    summary,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
		},
	})
}

// GetLabDiscussions handles GET /feedback/{labId}/discussions
func (h *FeedbackHandler) GetLabDiscussions(c *gin.Context) {
	labIDStr := c.Param("labId")
	labID, err := strconv.ParseInt(labIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid lab ID"})
		return
	}
	
	page, limit := middleware.GetPaginationParams(c)
	
	discussions, err := h.service.GetLabDiscussions(labID, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get discussions", "details": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"discussions": discussions,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
		},
	})
}

// CreateDiscussion handles POST /feedback/discussions/create
func (h *FeedbackHandler) CreateDiscussion(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not provided by API Gateway"})
		return
	}
	
	var req models.DiscussionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}
	
	discussion, err := h.service.CreateDiscussion(userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create discussion", "details": err.Error()})
		return
	}
	
	c.JSON(http.StatusCreated, gin.H{
		"message":    "Discussion created successfully",
		"discussion": discussion,
	})
}

// CreateDiscussionReply handles POST /feedback/discussions/{id}/reply
func (h *FeedbackHandler) CreateDiscussionReply(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not provided by API Gateway"})
		return
	}
	
	discussionIDStr := c.Param("id")
	discussionID, err := strconv.ParseInt(discussionIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid discussion ID"})
		return
	}
	
	var req models.DiscussionReplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}
	
	// Ensure the discussion ID from URL matches the request
	req.DiscussionID = discussionID
	
	reply, err := h.service.CreateDiscussionReply(userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create reply", "details": err.Error()})
		return
	}
	
	c.JSON(http.StatusCreated, gin.H{
		"message": "Reply created successfully",
		"reply":   reply,
	})
}

// GetDiscussionReplies handles GET /feedback/discussions/{id}/replies
func (h *FeedbackHandler) GetDiscussionReplies(c *gin.Context) {
	discussionIDStr := c.Param("id")
	discussionID, err := strconv.ParseInt(discussionIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid discussion ID"})
		return
	}
	
	replies, err := h.service.GetDiscussionReplies(discussionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get replies", "details": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"replies": replies,
	})
}

// GetPendingReviews handles GET /feedback/pending
func (h *FeedbackHandler) GetPendingReviews(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not provided by API Gateway"})
		return
	}
	
	page, limit := middleware.GetPaginationParams(c)
	
	pending, err := h.service.GetPendingReviews(userID, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get pending reviews", "details": err.Error()})
		return
	}
	
	if len(pending) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"message": "No submissions available for review at the moment. You can review submissions from labs you've authored or completed.",
			"pending_reviews": []models.PendingReview{},
			"pagination": gin.H{
				"page":  page,
				"limit": limit,
			},
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"pending_reviews": pending,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
		},
	})
}

// GetUserStats handles GET /feedback/stats
func (h *FeedbackHandler) GetUserStats(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not provided by API Gateway"})
		return
	}
	
	stats, err := h.service.GetUserReviewStats(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user stats", "details": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"stats": stats,
	})
}
