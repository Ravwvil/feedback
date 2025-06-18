package grpc

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log"

	"github.com/Ravwvil/feedback/internal/grpc/proto"
	"github.com/Ravwvil/feedback/internal/service"
)

type FeedbackGRPCServer struct {
	proto.UnimplementedFeedbackServiceServer
	feedbackService *service.FeedbackService
}

func NewFeedbackGRPCServer(feedbackService *service.FeedbackService) *FeedbackGRPCServer {
	return &FeedbackGRPCServer{
		feedbackService: feedbackService,
	}
}

func (s *FeedbackGRPCServer) CreateFeedback(ctx context.Context, req *proto.CreateFeedbackRequest) (*proto.CreateFeedbackResponse, error) {
	// Calculate content hash
	contentHash := fmt.Sprintf("%x", sha256.Sum256([]byte(req.Content)))
	
	feedback, err := s.feedbackService.CreateFeedback(ctx, &service.CreateFeedbackParams{
		UserID:      req.UserId,
		LabID:       req.LabId,
		Title:       req.Title,
		Content:     req.Content,
		ContentHash: contentHash,
	})
	if err != nil {
		log.Printf("Failed to create feedback: %v", err)
		return nil, err
	}

	return &proto.CreateFeedbackResponse{
		Feedback: &proto.FeedbackFile{
			Id:          feedback.ID,
			UserId:      feedback.UserID,
			LabId:       feedback.LabID,
			Title:       feedback.Title,
			Content:     feedback.Content,
			ContentHash: feedback.ContentHash,
			CreatedAt:   feedback.CreatedAt.Unix(),
			UpdatedAt:   feedback.UpdatedAt.Unix(),
		},
	}, nil
}

func (s *FeedbackGRPCServer) GetFeedback(ctx context.Context, req *proto.GetFeedbackRequest) (*proto.GetFeedbackResponse, error) {
	feedback, err := s.feedbackService.GetFeedback(ctx, req.Id)
	if err != nil {
		log.Printf("Failed to get feedback: %v", err)
		return nil, err
	}

	return &proto.GetFeedbackResponse{
		Feedback: &proto.FeedbackFile{
			Id:          feedback.ID,
			UserId:      feedback.UserID,
			LabId:       feedback.LabID,
			Title:       feedback.Title,
			Content:     feedback.Content,
			ContentHash: feedback.ContentHash,
			CreatedAt:   feedback.CreatedAt.Unix(),
			UpdatedAt:   feedback.UpdatedAt.Unix(),
		},
	}, nil
}

func (s *FeedbackGRPCServer) UpdateFeedback(ctx context.Context, req *proto.UpdateFeedbackRequest) (*proto.UpdateFeedbackResponse, error) {
	var contentHash string
	if req.Content != "" {
		contentHash = fmt.Sprintf("%x", sha256.Sum256([]byte(req.Content)))
	}

	feedback, err := s.feedbackService.UpdateFeedback(ctx, &service.UpdateFeedbackParams{
		ID:          req.Id,
		Title:       req.Title,
		Content:     req.Content,
		ContentHash: contentHash,
	})
	if err != nil {
		log.Printf("Failed to update feedback: %v", err)
		return nil, err
	}

	return &proto.UpdateFeedbackResponse{
		Feedback: &proto.FeedbackFile{
			Id:          feedback.ID,
			UserId:      feedback.UserID,
			LabId:       feedback.LabID,
			Title:       feedback.Title,
			Content:     feedback.Content,
			ContentHash: feedback.ContentHash,
			CreatedAt:   feedback.CreatedAt.Unix(),
			UpdatedAt:   feedback.UpdatedAt.Unix(),
		},
	}, nil
}

func (s *FeedbackGRPCServer) DeleteFeedback(ctx context.Context, req *proto.DeleteFeedbackRequest) (*proto.DeleteFeedbackResponse, error) {
	err := s.feedbackService.DeleteFeedback(ctx, req.Id)
	if err != nil {
		log.Printf("Failed to delete feedback: %v", err)
		return nil, err
	}

	return &proto.DeleteFeedbackResponse{
		Success: true,
	}, nil
}

func (s *FeedbackGRPCServer) ListUserFeedbacks(ctx context.Context, req *proto.ListUserFeedbacksRequest) (*proto.ListUserFeedbacksResponse, error) {
	feedbacks, totalCount, err := s.feedbackService.ListUserFeedbacks(ctx, &service.ListUserFeedbacksParams{
		UserID: req.UserId,
		LabID:  req.LabId,
		Page:   int(req.Page),
		Limit:  int(req.Limit),
	})
	if err != nil {
		log.Printf("Failed to list user feedbacks: %v", err)
		return nil, err
	}

	protoFeedbacks := make([]*proto.FeedbackFile, len(feedbacks))
	for i, feedback := range feedbacks {
		protoFeedbacks[i] = &proto.FeedbackFile{
			Id:          feedback.ID,
			UserId:      feedback.UserID,
			LabId:       feedback.LabID,
			Title:       feedback.Title,
			Content:     feedback.Content,
			ContentHash: feedback.ContentHash,
			CreatedAt:   feedback.CreatedAt.Unix(),
			UpdatedAt:   feedback.UpdatedAt.Unix(),
		}
	}

	return &proto.ListUserFeedbacksResponse{
		Feedbacks:  protoFeedbacks,
		TotalCount: int32(totalCount),
	}, nil
}

func (s *FeedbackGRPCServer) UploadAsset(stream proto.FeedbackService_UploadAssetServer) error {
	var feedbackID, filename, contentType string
	var totalSize int64
	var buffer []byte

	// Receive first message with metadata
	req, err := stream.Recv()
	if err != nil {
		return err
	}

	if metadata := req.GetMetadata(); metadata != nil {
		feedbackID = metadata.FeedbackId
		filename = metadata.Filename
		contentType = metadata.ContentType
		totalSize = metadata.TotalSize
		buffer = make([]byte, 0, totalSize)
	} else {
		return fmt.Errorf("first message must contain metadata")
	}

	// Receive file chunks
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if chunk := req.GetChunk(); chunk != nil {
			buffer = append(buffer, chunk...)
		}
	}

	// Upload to MinIO
	size, err := s.feedbackService.UploadAsset(context.Background(), feedbackID, filename, contentType, buffer)
	if err != nil {
		log.Printf("Failed to upload asset: %v", err)
		return err
	}

	return stream.SendAndClose(&proto.UploadAssetResponse{
		Filename: filename,
		Size:     size,
		Success:  true,
	})
}

func (s *FeedbackGRPCServer) DownloadAsset(req *proto.DownloadAssetRequest, stream proto.FeedbackService_DownloadAssetServer) error {
	assetInfo, data, err := s.feedbackService.DownloadAsset(context.Background(), req.FeedbackId, req.Filename)
	if err != nil {
		log.Printf("Failed to download asset: %v", err)
		return err
	}

	// Send asset info first
	err = stream.Send(&proto.DownloadAssetResponse{
		Data: &proto.DownloadAssetResponse_Info{
			Info: &proto.AssetInfo{
				Filename:    assetInfo.Filename,
				Size:        assetInfo.Size,
				ContentType: assetInfo.ContentType,
				UploadedAt:  assetInfo.UploadedAt.Unix(),
			},
		},
	})
	if err != nil {
		return err
	}

	// Send file data in chunks
	chunkSize := 1024 * 64 // 64KB chunks
	for i := 0; i < len(data); i += chunkSize {
		end := i + chunkSize
		if end > len(data) {
			end = len(data)
		}

		err = stream.Send(&proto.DownloadAssetResponse{
			Data: &proto.DownloadAssetResponse_Chunk{
				Chunk: data[i:end],
			},
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *FeedbackGRPCServer) ListAssets(ctx context.Context, req *proto.ListAssetsRequest) (*proto.ListAssetsResponse, error) {
	assets, err := s.feedbackService.ListAssets(ctx, req.FeedbackId)
	if err != nil {
		log.Printf("Failed to list assets: %v", err)
		return nil, err
	}

	protoAssets := make([]*proto.AssetInfo, len(assets))
	for i, asset := range assets {
		protoAssets[i] = &proto.AssetInfo{
			Filename:    asset.Filename,
			Size:        asset.Size,
			ContentType: asset.ContentType,
			UploadedAt:  asset.UploadedAt.Unix(),
		}
	}

	return &proto.ListAssetsResponse{
		Assets: protoAssets,
	}, nil
}
