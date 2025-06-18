package main

import (
	"log"
	"net"

	"google.golang.org/grpc"
	
	"github.com/Ravwvil/feedback/internal/config"
	"github.com/Ravwvil/feedback/internal/database"
	pb "github.com/Ravwvil/feedback/internal/grpc/proto"
	grpcServer "github.com/Ravwvil/feedback/internal/grpc"
	"github.com/Ravwvil/feedback/internal/repository"
	"github.com/Ravwvil/feedback/internal/service"
	"github.com/Ravwvil/feedback/internal/storage"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Connect to database
	db, err := database.NewConnection(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize repository
	feedbackRepo := repository.NewFeedbackRepository(db)

	// Initialize MinIO client
	minioClient, err := storage.NewMinIOClient(
		cfg.MinIOEndpoint,
		cfg.MinIOAccessKey,
		cfg.MinIOSecretKey,
		cfg.MinIOBucketName,
		cfg.MinIOUseSSL,
	)
	if err != nil {
		log.Fatalf("Failed to initialize MinIO client: %v", err)
	}

	// Initialize service
	feedbackService := service.NewFeedbackService(feedbackRepo, minioClient)

	// Initialize gRPC server
	grpcListener, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Fatalf("Failed to listen on gRPC port: %v", err)
	}

	grpcSrv := grpc.NewServer()
	feedbackGRPCServer := grpcServer.NewFeedbackGRPCServer(feedbackService)
	proto.RegisterFeedbackServiceServer(grpcSrv, feedbackGRPCServer)

	log.Printf("Starting Minimal Feedback Service (gRPC) on port %s", cfg.GRPCPort)
	log.Printf("Database: %s:%s/%s", cfg.DBHost, cfg.DBPort, cfg.DBName)
	log.Printf("MinIO: %s/%s", cfg.MinIOEndpoint, cfg.MinIOBucketName)
	
	if err := grpcSrv.Serve(grpcListener); err != nil {
		log.Fatalf("Failed to start gRPC server: %v", err)
	}
}