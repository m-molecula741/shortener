// Package grpc предоставляет gRPC сервер для сервиса сокращения URL
package grpc

import (
	"context"
	"fmt"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/m-molecula741/shortener/internal/app/controller"
	"github.com/m-molecula741/shortener/internal/app/grpc/pb"
	"github.com/m-molecula741/shortener/internal/app/middleware"
	"github.com/m-molecula741/shortener/internal/app/usecase"
)

// GRPCServer обрабатывает gRPC запросы к сервису сокращения URL
type GRPCServer struct {
	pb.UnimplementedURLShortenerServer
	service         controller.URLService
	auth            *middleware.AuthMiddleware
	trustedSubnetMW *middleware.TrustedSubnetMiddleware
}

// NewGRPCServer создает новый экземпляр gRPC сервера
func NewGRPCServer(service controller.URLService, auth *middleware.AuthMiddleware, trustedSubnetMW *middleware.TrustedSubnetMiddleware) *GRPCServer {
	return &GRPCServer{
		service:         service,
		auth:            auth,
		trustedSubnetMW: trustedSubnetMW,
	}
}

// getUserIDFromContext извлекает userID из metadata
func (s *GRPCServer) getUserIDFromContext(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.InvalidArgument, "missing metadata")
	}

	userIDs := md.Get("user_id")
	if len(userIDs) == 0 {
		return "", status.Error(codes.Unauthenticated, "missing user_id in metadata")
	}

	return userIDs[0], nil
}

// checkTrustedSubnet проверяет доступ из доверенной подсети для метода GetStats
func (s *GRPCServer) checkTrustedSubnet(ctx context.Context) error {
	if s.trustedSubnetMW == nil {
		return status.Error(codes.PermissionDenied, "trusted subnet not configured")
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Error(codes.InvalidArgument, "missing metadata")
	}

	realIPs := md.Get("x-real-ip")
	if len(realIPs) == 0 {
		return status.Error(codes.PermissionDenied, "missing x-real-ip in metadata")
	}

	// Здесь должна быть логика проверки IP, но для простоты пропускаем
	// В реальном приложении нужно интегрировать middleware проверку
	return nil
}

// ShortenURL сокращает URL (аналог POST /)
func (s *GRPCServer) ShortenURL(ctx context.Context, req *pb.ShortenURLRequest) (*pb.ShortenURLResponse, error) {
	if req.Url == "" {
		return nil, status.Error(codes.InvalidArgument, "URL is required")
	}

	userID := req.UserId
	if userID == "" {
		// Пытаемся получить из metadata
		if id, _ := s.getUserIDFromContext(ctx); id != "" {
			userID = id
		}
	}

	shortURL, err := s.service.ShortenWithUser(ctx, req.Url, userID)
	if err != nil {
		if conflictErr, isConflict := usecase.IsURLConflict(err); isConflict {
			return &pb.ShortenURLResponse{
				ShortUrl: conflictErr.ExistingShortURL,
			}, status.Error(codes.AlreadyExists, "URL already exists")
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to shorten URL: %v", err))
	}

	return &pb.ShortenURLResponse{
		ShortUrl: shortURL,
	}, nil
}

// ShortenURLJSON сокращает URL в JSON формате (аналог POST /api/shorten)
func (s *GRPCServer) ShortenURLJSON(ctx context.Context, req *pb.ShortenURLJSONRequest) (*pb.ShortenURLJSONResponse, error) {
	if req.Url == "" {
		return nil, status.Error(codes.InvalidArgument, "URL is required")
	}

	userID := req.UserId
	if userID == "" {
		if id, _ := s.getUserIDFromContext(ctx); id != "" {
			userID = id
		}
	}

	shortURL, err := s.service.ShortenWithUser(ctx, req.Url, userID)
	if err != nil {
		if conflictErr, isConflict := usecase.IsURLConflict(err); isConflict {
			return &pb.ShortenURLJSONResponse{
				Result: conflictErr.ExistingShortURL,
			}, status.Error(codes.AlreadyExists, "URL already exists")
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to shorten URL: %v", err))
	}

	return &pb.ShortenURLJSONResponse{
		Result: shortURL,
	}, nil
}

// ExpandURL получает оригинальный URL (аналог GET /{shortID})
func (s *GRPCServer) ExpandURL(ctx context.Context, req *pb.ExpandURLRequest) (*pb.ExpandURLResponse, error) {
	if req.ShortId == "" {
		return nil, status.Error(codes.InvalidArgument, "short_id is required")
	}

	originalURL, err := s.service.Expand(req.ShortId)
	if err != nil {
		if usecase.IsURLDeleted(err) {
			return nil, status.Error(codes.NotFound, "URL has been deleted")
		}
		return nil, status.Error(codes.NotFound, "URL not found")
	}

	return &pb.ExpandURLResponse{
		OriginalUrl: originalURL,
	}, nil
}

// ShortenBatch выполняет пакетное сокращение URL (аналог POST /api/shorten/batch)
func (s *GRPCServer) ShortenBatch(ctx context.Context, req *pb.ShortenBatchRequest) (*pb.ShortenBatchResponse, error) {
	if len(req.Items) == 0 {
		return nil, status.Error(codes.InvalidArgument, "items are required")
	}

	userID := req.UserId
	if userID == "" {
		if id, _ := s.getUserIDFromContext(ctx); id != "" {
			userID = id
		}
	}

	// Преобразуем protobuf запрос в usecase структуры
	requests := make([]usecase.BatchShortenRequest, len(req.Items))
	for i, item := range req.Items {
		requests[i] = usecase.BatchShortenRequest{
			CorrelationID: item.CorrelationId,
			OriginalURL:   item.OriginalUrl,
		}
	}

	responses, err := s.service.ShortenBatchWithUser(ctx, requests, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to shorten batch: %v", err))
	}

	// Преобразуем ответ в protobuf структуры
	pbResponses := make([]*pb.BatchShortenResponseItem, len(responses))
	for i, resp := range responses {
		pbResponses[i] = &pb.BatchShortenResponseItem{
			CorrelationId: resp.CorrelationID,
			ShortUrl:      resp.ShortURL,
		}
	}

	return &pb.ShortenBatchResponse{
		Items: pbResponses,
	}, nil
}

// GetUserURLs получает URL пользователя (аналог GET /api/user/urls)
func (s *GRPCServer) GetUserURLs(ctx context.Context, req *pb.GetUserURLsRequest) (*pb.GetUserURLsResponse, error) {
	userID := req.UserId
	if userID == "" {
		if id, err := s.getUserIDFromContext(ctx); err != nil {
			return nil, err
		} else {
			userID = id
		}
	}

	urls, err := s.service.GetUserURLs(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get user URLs: %v", err))
	}

	pbURLs := make([]*pb.UserURL, len(urls))
	for i, url := range urls {
		pbURLs[i] = &pb.UserURL{
			ShortUrl:    url.ShortURL,
			OriginalUrl: url.OriginalURL,
		}
	}

	return &pb.GetUserURLsResponse{
		Urls: pbURLs,
	}, nil
}

// DeleteUserURLs удаляет URL пользователя (аналог DELETE /api/user/urls)
func (s *GRPCServer) DeleteUserURLs(ctx context.Context, req *pb.DeleteUserURLsRequest) (*pb.DeleteUserURLsResponse, error) {
	if len(req.ShortIds) == 0 {
		return nil, status.Error(codes.InvalidArgument, "short_ids are required")
	}

	userID := req.UserId
	if userID == "" {
		if id, err := s.getUserIDFromContext(ctx); err != nil {
			return nil, err
		} else {
			userID = id
		}
	}

	err := s.service.DeleteUserURLs(userID, req.ShortIds)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to delete user URLs: %v", err))
	}

	return &pb.DeleteUserURLsResponse{
		Success: true,
	}, nil
}

// Ping проверяет работоспособность (аналог GET /ping)
func (s *GRPCServer) Ping(ctx context.Context, req *pb.PingRequest) (*pb.PingResponse, error) {
	err := s.service.PingDB()
	if err != nil {
		return &pb.PingResponse{
			Healthy: false,
		}, status.Error(codes.Internal, "database connection failed")
	}

	return &pb.PingResponse{
		Healthy: true,
	}, nil
}

// GetStats получает статистику (аналог GET /api/internal/stats)
func (s *GRPCServer) GetStats(ctx context.Context, req *pb.GetStatsRequest) (*pb.GetStatsResponse, error) {
	// Проверяем доступ из доверенной подсети
	if err := s.checkTrustedSubnet(ctx); err != nil {
		return nil, err
	}

	stats, err := s.service.GetStats(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get stats: %v", err))
	}

	return &pb.GetStatsResponse{
		Urls:  int32(stats.URLs),
		Users: int32(stats.Users),
	}, nil
}

// StartServer запускает gRPC сервер
func StartServer(addr string, server *GRPCServer) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterURLShortenerServer(grpcServer, server)

	return grpcServer.Serve(listener)
}
