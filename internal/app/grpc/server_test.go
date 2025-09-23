package grpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"

	"github.com/m-molecula741/shortener/internal/app/grpc/pb"
	"github.com/m-molecula741/shortener/internal/app/middleware"
	"github.com/m-molecula741/shortener/internal/app/usecase"
)

// MockURLService для тестирования gRPC сервера
type MockURLService struct {
	ShortenWithUserFunc func(ctx context.Context, url, userID string) (string, error)
	ExpandFunc          func(shortID string) (string, error)
	PingDBFunc          func() error
	GetStatsFunc        func(ctx context.Context) (usecase.Stats, error)
}

func (m *MockURLService) Shorten(url string) (string, error) {
	return "http://localhost:8080/abc123", nil
}

func (m *MockURLService) ShortenWithUser(ctx context.Context, url, userID string) (string, error) {
	if m.ShortenWithUserFunc != nil {
		return m.ShortenWithUserFunc(ctx, url, userID)
	}
	return "http://localhost:8080/abc123", nil
}

func (m *MockURLService) Expand(shortID string) (string, error) {
	if m.ExpandFunc != nil {
		return m.ExpandFunc(shortID)
	}
	return "http://example.com", nil
}

func (m *MockURLService) PingDB() error {
	if m.PingDBFunc != nil {
		return m.PingDBFunc()
	}
	return nil
}

func (m *MockURLService) ShortenBatch(ctx context.Context, requests []usecase.BatchShortenRequest) ([]usecase.BatchShortenResponse, error) {
	return nil, nil
}

func (m *MockURLService) ShortenBatchWithUser(ctx context.Context, requests []usecase.BatchShortenRequest, userID string) ([]usecase.BatchShortenResponse, error) {
	responses := make([]usecase.BatchShortenResponse, len(requests))
	for i, req := range requests {
		responses[i] = usecase.BatchShortenResponse{
			CorrelationID: req.CorrelationID,
			ShortURL:      "http://localhost:8080/batch" + string(rune(i+'1')),
		}
	}
	return responses, nil
}

func (m *MockURLService) GetUserURLs(ctx context.Context, userID string) ([]usecase.UserURL, error) {
	return []usecase.UserURL{
		{
			ShortURL:    "http://localhost:8080/abc123",
			OriginalURL: "http://example.com",
		},
	}, nil
}

func (m *MockURLService) DeleteUserURLs(userID string, shortIDs []string) error {
	return nil
}

func (m *MockURLService) GetStats(ctx context.Context) (usecase.Stats, error) {
	if m.GetStatsFunc != nil {
		return m.GetStatsFunc(ctx)
	}
	return usecase.Stats{URLs: 10, Users: 5}, nil
}

func createTestGRPCServer() *GRPCServer {
	mockService := &MockURLService{}
	auth, _ := middleware.NewAuthMiddleware("test-key")
	trustedSubnetMW, _ := middleware.NewTrustedSubnetMiddleware("192.168.1.0/24")
	return NewGRPCServer(mockService, auth, trustedSubnetMW)
}

func TestGRPCServer_ShortenURL(t *testing.T) {
	server := createTestGRPCServer()

	req := &pb.ShortenURLRequest{
		Url:    "http://example.com",
		UserId: "test-user",
	}

	resp, err := server.ShortenURL(context.Background(), req)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.ShortUrl)
	assert.Equal(t, "http://localhost:8080/abc123", resp.ShortUrl)
}

func TestGRPCServer_ExpandURL(t *testing.T) {
	server := createTestGRPCServer()

	req := &pb.ExpandURLRequest{
		ShortId: "abc123",
	}

	resp, err := server.ExpandURL(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "http://example.com", resp.OriginalUrl)
}

func TestGRPCServer_Ping(t *testing.T) {
	server := createTestGRPCServer()

	req := &pb.PingRequest{}

	resp, err := server.Ping(context.Background(), req)
	require.NoError(t, err)
	assert.True(t, resp.Healthy)
}

func TestGRPCServer_GetStats(t *testing.T) {
	server := createTestGRPCServer()

	// Создаем контекст с метаданными для доверенной подсети
	md := metadata.New(map[string]string{
		"x-real-ip": "192.168.1.100",
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	req := &pb.GetStatsRequest{}

	resp, err := server.GetStats(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, int32(10), resp.Urls)
	assert.Equal(t, int32(5), resp.Users)
}

func TestGRPCServer_ShortenBatch(t *testing.T) {
	server := createTestGRPCServer()

	req := &pb.ShortenBatchRequest{
		Items: []*pb.BatchShortenItem{
			{
				CorrelationId: "1",
				OriginalUrl:   "http://example1.com",
			},
			{
				CorrelationId: "2",
				OriginalUrl:   "http://example2.com",
			},
		},
		UserId: "test-user",
	}

	resp, err := server.ShortenBatch(context.Background(), req)
	require.NoError(t, err)
	assert.Len(t, resp.Items, 2)
	assert.Equal(t, "1", resp.Items[0].CorrelationId)
	assert.Equal(t, "2", resp.Items[1].CorrelationId)
}
