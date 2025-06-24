package singbox

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/highlight-apps/node-backend/assets"
	"github.com/highlight-apps/node-backend/backend/common"
	"github.com/highlight-apps/node-backend/backend/singbox/api"
	"github.com/highlight-apps/node-backend/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func loadStatsConfig() (string, error) {
	path := common.DefaultSingboxConfigPath
	data, err := assets.ConfigFS.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("%w (%s): %w", common.ErrFailedReadEmbeddedConfig, path, err)
	}
	return string(data), nil
}

func loadConfigWithUsers() string {
	return `{
		"log": {
			"level": "info"
		},
		"inbounds": [
			{
				"type": "vmess",
				"tag": "test-inbound",
				"listen": "127.0.0.1",
				"listen_port": 7891,
				"users": [
					{
						"uuid": "12345678-1234-1234-1234-123456789abc",
						"name": "test-user"
					}
				],
				"transport": {
					"type": "ws",
					"path": "/test"
				}
			}
		],
		"outbounds": [
			{
				"type": "direct",
				"tag": "test-outbound"
			}
		],
		"experimental": {
			"v2ray_api": {
				"listen": "127.0.0.1:8080",
				"stats": {
					"enabled": true,
					"users": ["test-user"],
					"inbounds": ["test-inbound"],
					"outbounds": ["test-outbound"]
				}
			}
		}
	}`
}

type MockStatsServiceClient struct {
	getSysStatsFunc func(ctx context.Context, req *api.SysStatsRequest, opts ...grpc.CallOption) (*api.SysStatsResponse, error)
	queryStatsFunc  func(ctx context.Context, req *api.QueryStatsRequest, opts ...grpc.CallOption) (*api.QueryStatsResponse, error)
	getStatsFunc    func(ctx context.Context, req *api.GetStatsRequest, opts ...grpc.CallOption) (*api.GetStatsResponse, error)
}

func (m *MockStatsServiceClient) GetStats(ctx context.Context, req *api.GetStatsRequest, opts ...grpc.CallOption) (*api.GetStatsResponse, error) {
	if m.getStatsFunc != nil {
		return m.getStatsFunc(ctx, req, opts...)
	}
	return &api.GetStatsResponse{}, nil
}

func (m *MockStatsServiceClient) GetSysStats(ctx context.Context, req *api.SysStatsRequest, opts ...grpc.CallOption) (*api.SysStatsResponse, error) {
	if m.getSysStatsFunc != nil {
		return m.getSysStatsFunc(ctx, req, opts...)
	}
	return &api.SysStatsResponse{NumGoroutine: 10, Alloc: 1000, Uptime: 100}, nil
}

func (m *MockStatsServiceClient) QueryStats(ctx context.Context, req *api.QueryStatsRequest, opts ...grpc.CallOption) (*api.QueryStatsResponse, error) {
	if m.queryStatsFunc != nil {
		return m.queryStatsFunc(ctx, req, opts...)
	}

	var stats []*api.Stat
	if strings.Contains(req.Pattern, "malformed") {
		stats = append(stats, &api.Stat{Name: "invalid", Value: 100})
	} else {
		stats = append(stats, &api.Stat{Name: "user>>>test-user>>>traffic>>>uplink", Value: 1000})
		stats = append(stats, &api.Stat{Name: "user>>>test-user>>>traffic>>>downlink", Value: 2000})
	}

	return &api.QueryStatsResponse{Stat: stats}, nil
}

func waitForPortToListen(host string, port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), time.Second)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("port %s:%d not listening after %v", host, port, timeout)
}

func newTestStatsRunner(t *testing.T) *SingboxRunner {
	tl := logging.NewStdLogger()
	exePath, err := exec.LookPath(common.DefaultSingboxExecutablePath)
	if err != nil {
		t.Skipf("failed to find singbox executable: %v", err)
	}

	runner, err := NewSingboxRunner(exePath, tl)
	if err != nil {
		t.Fatalf("failed to create runner: %v", err)
	}
	return runner
}

func setupRealSingBox(t *testing.T) (*SingBoxAPI, func()) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	runner := newTestStatsRunner(t)

	config, err := loadStatsConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	t.Logf("Using config:\n%s", config)

	err = runner.Start(config)
	if err != nil {
		t.Fatalf("Failed to start SingBox: %v", err)
	}

	if err := waitForPortToListen("127.0.0.1", 8080, 10*time.Second); err != nil {
		runner.Stop()
		t.Fatalf("gRPC API not available: %v", err)
	}

	api, err := NewSingBoxAPI("127.0.0.1", 8080)
	if err != nil {
		runner.Stop()
		t.Fatalf("Failed to create API client: %v", err)
	}

	return api, func() {
		if api != nil {
			api.Close()
		}
		runner.Stop()
	}
}

func setupRealSingBoxWithUsers(t *testing.T) (*SingBoxAPI, func()) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	runner := newTestStatsRunner(t)
	config := loadConfigWithUsers()

	t.Logf("Using config with users:\n%s", config)

	err := runner.Start(config)
	if err != nil {
		t.Fatalf("Failed to start SingBox: %v", err)
	}

	if err := waitForPortToListen("127.0.0.1", 8080, 10*time.Second); err != nil {
		runner.Stop()
		t.Fatalf("gRPC API not available: %v", err)
	}

	api, err := NewSingBoxAPI("127.0.0.1", 8080)
	if err != nil {
		runner.Stop()
		t.Fatalf("Failed to create API client: %v", err)
	}

	return api, func() {
		if api != nil {
			api.Close()
		}
		runner.Stop()
	}
}

func createMockAPI() *SingBoxAPI {
	conn, _ := grpc.NewClient("localhost:9999", grpc.WithTransportCredentials(insecure.NewCredentials()))
	return &SingBoxAPI{
		SingBoxAPIBase: &SingBoxAPIBase{
			address: "localhost",
			port:    9999,
			conn:    conn,
		},
		client: &MockStatsServiceClient{},
	}
}

func TestNewSingBoxAPIBase_Success(t *testing.T) {
	api, cleanup := setupRealSingBox(t)
	defer cleanup()

	base := api.SingBoxAPIBase
	if base.address != "127.0.0.1" {
		t.Errorf("Expected address '127.0.0.1', got '%s'", base.address)
	}
	if base.port != 8080 {
		t.Errorf("Expected port 8080, got %d", base.port)
	}
	if base.conn == nil {
		t.Error("Expected non-nil connection")
	}
}

func TestNewSingBoxAPIBase_DialError(t *testing.T) {
	base, err := NewSingBoxAPIBase("invalid-host-that-does-not-exist.local", 99999)
	if err != nil {
		return
	}
	defer base.Close()

	client := api.NewStatsServiceClient(base.conn)
	_, err = client.GetSysStats(context.Background(), &api.SysStatsRequest{})
	if err == nil {
		t.Error("Expected error for invalid host")
	}
}

func TestNewSingBoxAPIBase_EmptyAddress(t *testing.T) {
	base, err := NewSingBoxAPIBase("", 8080)
	if err != nil {
		return
	}
	if base != nil {
		defer base.Close()
		if base.address != "" {
			t.Errorf("Expected empty address, got '%s'", base.address)
		}
		if base.port != 8080 {
			t.Errorf("Expected port 8080, got %d", base.port)
		}
	}
}

func TestNewSingBoxAPIBase_NegativePort(t *testing.T) {
	base, err := NewSingBoxAPIBase("127.0.0.1", -1)
	if err != nil {
		return
	}
	if base != nil {
		defer base.Close()
		if base.address != "127.0.0.1" {
			t.Errorf("Expected address '127.0.0.1', got '%s'", base.address)
		}
		if base.port != -1 {
			t.Errorf("Expected port -1, got %d", base.port)
		}
	}
}

func TestNewSingBoxAPIBase_ZeroPort(t *testing.T) {
	base, err := NewSingBoxAPIBase("127.0.0.1", 0)
	if err != nil {
		return
	}
	if base != nil {
		defer base.Close()
		if base.address != "127.0.0.1" {
			t.Errorf("Expected address '127.0.0.1', got '%s'", base.address)
		}
		if base.port != 0 {
			t.Errorf("Expected port 0, got %d", base.port)
		}
	}
}

func TestNewSingBoxAPIBase_MaxPort(t *testing.T) {
	base, err := NewSingBoxAPIBase("127.0.0.1", 65535)
	if err != nil {
		return
	}
	if base != nil {
		defer base.Close()
		if base.address != "127.0.0.1" {
			t.Errorf("Expected address '127.0.0.1', got '%s'", base.address)
		}
		if base.port != 65535 {
			t.Errorf("Expected port 65535, got %d", base.port)
		}
	}
}

func TestNewSingBoxAPIBase_GRPCClientError(t *testing.T) {
	originalFactory := grpcClientFactory
	grpcClientFactory = func(target string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
		return nil, errors.New("mocked grpc client error")
	}
	defer func() {
		grpcClientFactory = originalFactory
	}()

	_, err := NewSingBoxAPIBase("127.0.0.1", 8080)
	if err == nil {
		t.Error("Expected error from mocked grpc client")
	}
	if err.Error() != "mocked grpc client error" {
		t.Errorf("Expected 'mocked grpc client error', got '%s'", err.Error())
	}
}

func TestSingBoxAPIBase_Close_Success(t *testing.T) {
	api, cleanup := setupRealSingBox(t)
	defer cleanup()

	err := api.Close()
	if err != nil {
		t.Errorf("Close() failed: %v", err)
	}
}

func TestSingBoxAPIBase_Close_WithError(t *testing.T) {
	conn, _ := grpc.NewClient("localhost:9999", grpc.WithTransportCredentials(insecure.NewCredentials()))
	base := &SingBoxAPIBase{
		address: "127.0.0.1",
		port:    8080,
		conn:    conn,
	}

	conn.Close()

	err := base.Close()
	if err == nil {
		t.Error("Expected close error")
	}
}

func TestSingBoxAPIBase_Close_NilConnection(t *testing.T) {
	base := &SingBoxAPIBase{
		address: "127.0.0.1",
		port:    8080,
		conn:    nil,
	}

	err := base.Close()
	if err != nil {
		t.Errorf("Close() with nil connection should not fail: %v", err)
	}
}

func TestNewSingBoxAPI_Success(t *testing.T) {
	api, cleanup := setupRealSingBox(t)
	defer cleanup()

	if api.SingBoxAPIBase == nil {
		t.Error("Expected non-nil SingBoxAPIBase")
	}
	if api.client == nil {
		t.Error("Expected non-nil client")
	}
}

func TestNewSingBoxAPI_BaseCreationError(t *testing.T) {
	sbAPI, err := NewSingBoxAPI("invalid-host-that-does-not-exist.local", 99999)
	if err != nil {
		return
	}
	if sbAPI != nil {
		defer sbAPI.Close()
		client := sbAPI.client
		_, err = client.GetSysStats(context.Background(), &api.SysStatsRequest{})
		if err == nil {
			t.Error("Expected error for invalid host")
		}
	}
}

func TestNewSingBoxAPI_EmptyAddress(t *testing.T) {
	sbAPI, err := NewSingBoxAPI("", 8080)
	if err != nil {
		return
	}
	if sbAPI != nil {
		defer sbAPI.Close()
		if sbAPI.address != "" {
			t.Errorf("Expected empty address, got '%s'", sbAPI.address)
		}
		if sbAPI.port != 8080 {
			t.Errorf("Expected port 8080, got %d", sbAPI.port)
		}
	}
}

func TestNewSingBoxAPI_NegativePort(t *testing.T) {
	sbAPI, err := NewSingBoxAPI("127.0.0.1", -1)
	if err != nil {
		return
	}
	if sbAPI != nil {
		defer sbAPI.Close()
		if sbAPI.address != "127.0.0.1" {
			t.Errorf("Expected address '127.0.0.1', got '%s'", sbAPI.address)
		}
		if sbAPI.port != -1 {
			t.Errorf("Expected port -1, got %d", sbAPI.port)
		}
	}
}

func TestNewSingBoxAPI_GRPCClientError(t *testing.T) {
	originalFactory := grpcClientFactory
	grpcClientFactory = func(target string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
		return nil, errors.New("mocked grpc client error")
	}
	defer func() {
		grpcClientFactory = originalFactory
	}()

	_, err := NewSingBoxAPI("127.0.0.1", 8080)
	if err == nil {
		t.Error("Expected error from mocked grpc client")
	}
	if err.Error() != "mocked grpc client error" {
		t.Errorf("Expected 'mocked grpc client error', got '%s'", err.Error())
	}
}

func TestSingBoxAPI_GetSysStats_Success(t *testing.T) {
	api, cleanup := setupRealSingBox(t)
	defer cleanup()

	ctx := context.Background()
	stats, err := api.GetSysStats(ctx)
	if err != nil {
		t.Fatalf("GetSysStats failed: %v", err)
	}

	if stats.NumGoroutine == 0 {
		t.Error("Expected NumGoroutine > 0")
	}
	if stats.Alloc == 0 {
		t.Error("Expected Alloc > 0")
	}
	if stats.Uptime == 0 {
		t.Error("Expected Uptime > 0")
	}

	t.Logf("SysStats: NumGoroutine=%d, Alloc=%d, Uptime=%d",
		stats.NumGoroutine, stats.Alloc, stats.Uptime)
}

func TestSingBoxAPI_GetSysStats_ContextCancellation(t *testing.T) {
	api, cleanup := setupRealSingBox(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := api.GetSysStats(ctx)
	if err == nil {
		t.Error("Expected error with cancelled context")
	}
}

func TestSingBoxAPI_GetSysStats_GRPCError(t *testing.T) {
	mockAPI := createMockAPI()
	mockAPI.client = &MockStatsServiceClient{
		getSysStatsFunc: func(ctx context.Context, req *api.SysStatsRequest, opts ...grpc.CallOption) (*api.SysStatsResponse, error) {
			return nil, fmt.Errorf("grpc error")
		},
	}

	_, err := mockAPI.GetSysStats(context.Background())
	if err == nil {
		t.Error("Expected gRPC error")
	}
}

func TestSingBoxAPI_queryStats_Success(t *testing.T) {
	api, cleanup := setupRealSingBox(t)
	defer cleanup()

	ctx := context.Background()

	patterns := []string{"user>>>", "inbound>>>", "outbound>>>"}
	for _, pattern := range patterns {
		stats, err := api.queryStats(ctx, pattern, false)
		if err != nil {
			t.Errorf("queryStats('%s') failed: %v", pattern, err)
		}
		t.Logf("Pattern '%s' returned %d stats", pattern, len(stats))
	}
}

func TestSingBoxAPI_queryStats_WithData(t *testing.T) {
	api, cleanup := setupRealSingBoxWithUsers(t)
	defer cleanup()

	ctx := context.Background()
	time.Sleep(2 * time.Second)

	patterns := []string{"user>>>", "inbound>>>", "outbound>>>"}
	for _, pattern := range patterns {
		stats, err := api.queryStats(ctx, pattern, false)
		if err != nil {
			t.Errorf("queryStats('%s') failed: %v", pattern, err)
		}
		t.Logf("Pattern '%s' returned %d stats", pattern, len(stats))

		for _, stat := range stats {
			if stat.Name == "" {
				t.Errorf("Expected non-empty Name in stat")
			}
		}
	}
}

func TestSingBoxAPI_queryStats_ContextCancellation(t *testing.T) {
	api, cleanup := setupRealSingBox(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := api.queryStats(ctx, "user>>>", false)
	if err == nil {
		t.Error("Expected error with cancelled context")
	}
}

func TestSingBoxAPI_queryStats_GRPCError(t *testing.T) {
	mockAPI := createMockAPI()
	mockAPI.client = &MockStatsServiceClient{
		queryStatsFunc: func(ctx context.Context, req *api.QueryStatsRequest, opts ...grpc.CallOption) (*api.QueryStatsResponse, error) {
			return nil, fmt.Errorf("grpc error")
		},
	}

	_, err := mockAPI.queryStats(context.Background(), "user>>>", false)
	if err == nil {
		t.Error("Expected gRPC error")
	}
}

func TestSingBoxAPI_queryStats_MalformedData(t *testing.T) {
	mockAPI := createMockAPI()
	mockAPI.client = &MockStatsServiceClient{
		queryStatsFunc: func(ctx context.Context, req *api.QueryStatsRequest, opts ...grpc.CallOption) (*api.QueryStatsResponse, error) {
			stats := []*api.Stat{
				{Name: "incomplete", Value: 100},
				{Name: "user>>>name", Value: 200},
				{Name: "user>>>name>>>traffic", Value: 300},
				{Name: "user>>>name>>>traffic>>>uplink", Value: 400},
			}
			return &api.QueryStatsResponse{Stat: stats}, nil
		},
	}

	results, err := mockAPI.queryStats(context.Background(), "user>>>", false)
	if err != nil {
		t.Fatalf("queryStats failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
}

func TestSingBoxAPI_queryStats_EmptyResponse(t *testing.T) {
	mockAPI := createMockAPI()
	mockAPI.client = &MockStatsServiceClient{
		queryStatsFunc: func(ctx context.Context, req *api.QueryStatsRequest, opts ...grpc.CallOption) (*api.QueryStatsResponse, error) {
			return &api.QueryStatsResponse{Stat: []*api.Stat{}}, nil
		},
	}

	results, err := mockAPI.queryStats(context.Background(), "user>>>", false)
	if err != nil {
		t.Fatalf("queryStats failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(results))
	}
}

func TestSingBoxAPI_GetUsersStats_Success(t *testing.T) {
	api, cleanup := setupRealSingBox(t)
	defer cleanup()

	ctx := context.Background()
	stats, err := api.GetUsersStats(ctx, false)
	if err != nil {
		t.Fatalf("GetUsersStats failed: %v", err)
	}

	t.Logf("Found %d user stats", len(stats))
	for _, stat := range stats {
		t.Logf("User stat: Name=%s, Type=%s, Link=%s, Value=%d",
			stat.Name, stat.Type, stat.Link, stat.Value)
	}
}

func TestSingBoxAPI_GetUsersStats_WithReset(t *testing.T) {
	api, cleanup := setupRealSingBoxWithUsers(t)
	defer cleanup()

	ctx := context.Background()

	stats1, err := api.GetUsersStats(ctx, false)
	if err != nil {
		t.Fatalf("GetUsersStats failed: %v", err)
	}

	stats2, err := api.GetUsersStats(ctx, true)
	if err != nil {
		t.Fatalf("GetUsersStats with reset failed: %v", err)
	}

	t.Logf("Stats without reset: %d, with reset: %d", len(stats1), len(stats2))
}

func TestSingBoxAPI_GetInboundsStats_Success(t *testing.T) {
	api, cleanup := setupRealSingBox(t)
	defer cleanup()

	ctx := context.Background()
	stats, err := api.GetInboundsStats(ctx, false)
	if err != nil {
		t.Fatalf("GetInboundsStats failed: %v", err)
	}

	t.Logf("Found %d inbound stats", len(stats))
	for _, stat := range stats {
		t.Logf("Inbound stat: Name=%s, Type=%s, Link=%s, Value=%d",
			stat.Name, stat.Type, stat.Link, stat.Value)
	}
}

func TestSingBoxAPI_GetInboundsStats_WithData(t *testing.T) {
	api, cleanup := setupRealSingBoxWithUsers(t)
	defer cleanup()

	ctx := context.Background()
	stats, err := api.GetInboundsStats(ctx, false)
	if err != nil {
		t.Fatalf("GetInboundsStats failed: %v", err)
	}

	t.Logf("Found %d inbound stats with data config", len(stats))
}

func TestSingBoxAPI_GetOutboundsStats_Success(t *testing.T) {
	api, cleanup := setupRealSingBox(t)
	defer cleanup()

	ctx := context.Background()
	stats, err := api.GetOutboundsStats(ctx, false)
	if err != nil {
		t.Fatalf("GetOutboundsStats failed: %v", err)
	}

	t.Logf("Found %d outbound stats", len(stats))
	for _, stat := range stats {
		t.Logf("Outbound stat: Name=%s, Type=%s, Link=%s, Value=%d",
			stat.Name, stat.Type, stat.Link, stat.Value)
	}
}

func TestSingBoxAPI_GetOutboundsStats_WithData(t *testing.T) {
	api, cleanup := setupRealSingBoxWithUsers(t)
	defer cleanup()

	ctx := context.Background()
	stats, err := api.GetOutboundsStats(ctx, false)
	if err != nil {
		t.Fatalf("GetOutboundsStats failed: %v", err)
	}

	t.Logf("Found %d outbound stats with data config", len(stats))
}

func TestSingBoxAPI_GetUserStats_Success(t *testing.T) {
	api, cleanup := setupRealSingBoxWithUsers(t)
	defer cleanup()

	ctx := context.Background()
	time.Sleep(2 * time.Second)

	stats, err := api.GetUserStats(ctx, "test-user", false)
	if err != nil {
		t.Fatalf("GetUserStats failed: %v", err)
	}

	if stats.Email != "test-user" {
		t.Errorf("Expected email 'test-user', got '%s'", stats.Email)
	}

	t.Logf("User 'test-user' stats: Uplink=%d, Downlink=%d",
		stats.Uplink, stats.Downlink)
}

func TestSingBoxAPI_GetUserStats_NonExistentUser(t *testing.T) {
	api, cleanup := setupRealSingBox(t)
	defer cleanup()

	ctx := context.Background()
	stats, err := api.GetUserStats(ctx, "non-existent-user", false)
	if err != nil {
		t.Fatalf("GetUserStats failed: %v", err)
	}

	if stats.Email != "non-existent-user" {
		t.Errorf("Expected email 'non-existent-user', got '%s'", stats.Email)
	}
	if stats.Uplink != 0 || stats.Downlink != 0 {
		t.Errorf("Expected zero stats for non-existent user, got Uplink=%d, Downlink=%d",
			stats.Uplink, stats.Downlink)
	}
}

func TestSingBoxAPI_GetUserStats_WithReset(t *testing.T) {
	api, cleanup := setupRealSingBoxWithUsers(t)
	defer cleanup()

	ctx := context.Background()

	stats, err := api.GetUserStats(ctx, "test-user", true)
	if err != nil {
		t.Fatalf("GetUserStats with reset failed: %v", err)
	}

	t.Logf("User stats with reset: Uplink=%d, Downlink=%d",
		stats.Uplink, stats.Downlink)
}

func TestSingBoxAPI_GetUserStats_ContextCancellation(t *testing.T) {
	api, cleanup := setupRealSingBox(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := api.GetUserStats(ctx, "test-user", false)
	if err == nil {
		t.Error("Expected error with cancelled context")
	}
}

func TestSingBoxAPI_GetUserStats_QueryStatsError(t *testing.T) {
	mockAPI := createMockAPI()
	mockAPI.client = &MockStatsServiceClient{
		queryStatsFunc: func(ctx context.Context, req *api.QueryStatsRequest, opts ...grpc.CallOption) (*api.QueryStatsResponse, error) {
			return nil, fmt.Errorf("query error")
		},
	}

	_, err := mockAPI.GetUserStats(context.Background(), "test-user", false)
	if err == nil {
		t.Error("Expected query error")
	}
}

func TestSingBoxAPI_GetUserStats_OnlyUplink(t *testing.T) {
	mockAPI := createMockAPI()
	mockAPI.client = &MockStatsServiceClient{
		queryStatsFunc: func(ctx context.Context, req *api.QueryStatsRequest, opts ...grpc.CallOption) (*api.QueryStatsResponse, error) {
			stats := []*api.Stat{
				{Name: "user>>>test-user>>>traffic>>>uplink", Value: 1000},
			}
			return &api.QueryStatsResponse{Stat: stats}, nil
		},
	}

	result, err := mockAPI.GetUserStats(context.Background(), "test-user", false)
	if err != nil {
		t.Fatalf("GetUserStats failed: %v", err)
	}

	if result.Uplink != 1000 {
		t.Errorf("Expected uplink 1000, got %d", result.Uplink)
	}
	if result.Downlink != 0 {
		t.Errorf("Expected downlink 0, got %d", result.Downlink)
	}
}

func TestSingBoxAPI_GetUserStats_OnlyDownlink(t *testing.T) {
	mockAPI := createMockAPI()
	mockAPI.client = &MockStatsServiceClient{
		queryStatsFunc: func(ctx context.Context, req *api.QueryStatsRequest, opts ...grpc.CallOption) (*api.QueryStatsResponse, error) {
			stats := []*api.Stat{
				{Name: "user>>>test-user>>>traffic>>>downlink", Value: 2000},
			}
			return &api.QueryStatsResponse{Stat: stats}, nil
		},
	}

	result, err := mockAPI.GetUserStats(context.Background(), "test-user", false)
	if err != nil {
		t.Fatalf("GetUserStats failed: %v", err)
	}

	if result.Uplink != 0 {
		t.Errorf("Expected uplink 0, got %d", result.Uplink)
	}
	if result.Downlink != 2000 {
		t.Errorf("Expected downlink 2000, got %d", result.Downlink)
	}
}

func TestSingBoxAPI_GetUserStats_UnknownLink(t *testing.T) {
	mockAPI := createMockAPI()
	mockAPI.client = &MockStatsServiceClient{
		queryStatsFunc: func(ctx context.Context, req *api.QueryStatsRequest, opts ...grpc.CallOption) (*api.QueryStatsResponse, error) {
			stats := []*api.Stat{
				{Name: "user>>>test-user>>>traffic>>>unknown", Value: 3000},
			}
			return &api.QueryStatsResponse{Stat: stats}, nil
		},
	}

	result, err := mockAPI.GetUserStats(context.Background(), "test-user", false)
	if err != nil {
		t.Fatalf("GetUserStats failed: %v", err)
	}

	if result.Uplink != 0 {
		t.Errorf("Expected uplink 0, got %d", result.Uplink)
	}
	if result.Downlink != 0 {
		t.Errorf("Expected downlink 0, got %d", result.Downlink)
	}
}

func TestSingBoxAPI_GetInboundStats_Success(t *testing.T) {
	api, cleanup := setupRealSingBoxWithUsers(t)
	defer cleanup()

	ctx := context.Background()
	time.Sleep(2 * time.Second)

	stats, err := api.GetInboundStats(ctx, "test-inbound", false)
	if err != nil {
		t.Fatalf("GetInboundStats failed: %v", err)
	}

	if stats.Tag != "test-inbound" {
		t.Errorf("Expected tag 'test-inbound', got '%s'", stats.Tag)
	}

	t.Logf("Inbound 'test-inbound' stats: Uplink=%d, Downlink=%d",
		stats.Uplink, stats.Downlink)
}

func TestSingBoxAPI_GetInboundStats_NonExistentInbound(t *testing.T) {
	api, cleanup := setupRealSingBox(t)
	defer cleanup()

	ctx := context.Background()
	stats, err := api.GetInboundStats(ctx, "non-existent-inbound", false)
	if err != nil {
		t.Fatalf("GetInboundStats failed: %v", err)
	}

	if stats.Tag != "non-existent-inbound" {
		t.Errorf("Expected tag 'non-existent-inbound', got '%s'", stats.Tag)
	}
	if stats.Uplink != 0 || stats.Downlink != 0 {
		t.Errorf("Expected zero stats for non-existent inbound, got Uplink=%d, Downlink=%d",
			stats.Uplink, stats.Downlink)
	}
}

func TestSingBoxAPI_GetInboundStats_WithReset(t *testing.T) {
	api, cleanup := setupRealSingBoxWithUsers(t)
	defer cleanup()

	ctx := context.Background()

	stats, err := api.GetInboundStats(ctx, "test-inbound", true)
	if err != nil {
		t.Fatalf("GetInboundStats with reset failed: %v", err)
	}

	t.Logf("Inbound stats with reset: Uplink=%d, Downlink=%d",
		stats.Uplink, stats.Downlink)
}

func TestSingBoxAPI_GetInboundStats_ContextCancellation(t *testing.T) {
	api, cleanup := setupRealSingBox(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := api.GetInboundStats(ctx, "test-inbound", false)
	if err == nil {
		t.Error("Expected error with cancelled context")
	}
}

func TestSingBoxAPI_GetInboundStats_QueryStatsError(t *testing.T) {
	mockAPI := createMockAPI()
	mockAPI.client = &MockStatsServiceClient{
		queryStatsFunc: func(ctx context.Context, req *api.QueryStatsRequest, opts ...grpc.CallOption) (*api.QueryStatsResponse, error) {
			return nil, fmt.Errorf("query error")
		},
	}

	_, err := mockAPI.GetInboundStats(context.Background(), "test-inbound", false)
	if err == nil {
		t.Error("Expected query error")
	}
}

func TestSingBoxAPI_GetInboundStats_OnlyUplink(t *testing.T) {
	mockAPI := createMockAPI()
	mockAPI.client = &MockStatsServiceClient{
		queryStatsFunc: func(ctx context.Context, req *api.QueryStatsRequest, opts ...grpc.CallOption) (*api.QueryStatsResponse, error) {
			stats := []*api.Stat{
				{Name: "inbound>>>test-inbound>>>traffic>>>uplink", Value: 1500},
			}
			return &api.QueryStatsResponse{Stat: stats}, nil
		},
	}

	result, err := mockAPI.GetInboundStats(context.Background(), "test-inbound", false)
	if err != nil {
		t.Fatalf("GetInboundStats failed: %v", err)
	}

	if result.Uplink != 1500 {
		t.Errorf("Expected uplink 1500, got %d", result.Uplink)
	}
	if result.Downlink != 0 {
		t.Errorf("Expected downlink 0, got %d", result.Downlink)
	}
}

func TestSingBoxAPI_GetInboundStats_OnlyDownlink(t *testing.T) {
	mockAPI := createMockAPI()
	mockAPI.client = &MockStatsServiceClient{
		queryStatsFunc: func(ctx context.Context, req *api.QueryStatsRequest, opts ...grpc.CallOption) (*api.QueryStatsResponse, error) {
			stats := []*api.Stat{
				{Name: "inbound>>>test-inbound>>>traffic>>>downlink", Value: 2500},
			}
			return &api.QueryStatsResponse{Stat: stats}, nil
		},
	}

	result, err := mockAPI.GetInboundStats(context.Background(), "test-inbound", false)
	if err != nil {
		t.Fatalf("GetInboundStats failed: %v", err)
	}

	if result.Uplink != 0 {
		t.Errorf("Expected uplink 0, got %d", result.Uplink)
	}
	if result.Downlink != 2500 {
		t.Errorf("Expected downlink 2500, got %d", result.Downlink)
	}
}

func TestSingBoxAPI_GetInboundStats_UnknownLink(t *testing.T) {
	mockAPI := createMockAPI()
	mockAPI.client = &MockStatsServiceClient{
		queryStatsFunc: func(ctx context.Context, req *api.QueryStatsRequest, opts ...grpc.CallOption) (*api.QueryStatsResponse, error) {
			stats := []*api.Stat{
				{Name: "inbound>>>test-inbound>>>traffic>>>unknown", Value: 3500},
			}
			return &api.QueryStatsResponse{Stat: stats}, nil
		},
	}

	result, err := mockAPI.GetInboundStats(context.Background(), "test-inbound", false)
	if err != nil {
		t.Fatalf("GetInboundStats failed: %v", err)
	}

	if result.Uplink != 0 {
		t.Errorf("Expected uplink 0, got %d", result.Uplink)
	}
	if result.Downlink != 0 {
		t.Errorf("Expected downlink 0, got %d", result.Downlink)
	}
}

func TestSingBoxAPI_GetOutboundStats_Success(t *testing.T) {
	api, cleanup := setupRealSingBoxWithUsers(t)
	defer cleanup()

	ctx := context.Background()
	time.Sleep(2 * time.Second)

	stats, err := api.GetOutboundStats(ctx, "test-outbound", false)
	if err != nil {
		t.Fatalf("GetOutboundStats failed: %v", err)
	}

	if stats.Tag != "test-outbound" {
		t.Errorf("Expected tag 'test-outbound', got '%s'", stats.Tag)
	}

	t.Logf("Outbound 'test-outbound' stats: Uplink=%d, Downlink=%d",
		stats.Uplink, stats.Downlink)
}

func TestSingBoxAPI_GetOutboundStats_NonExistentOutbound(t *testing.T) {
	api, cleanup := setupRealSingBox(t)
	defer cleanup()

	ctx := context.Background()
	stats, err := api.GetOutboundStats(ctx, "non-existent-outbound", false)
	if err != nil {
		t.Fatalf("GetOutboundStats failed: %v", err)
	}

	if stats.Tag != "non-existent-outbound" {
		t.Errorf("Expected tag 'non-existent-outbound', got '%s'", stats.Tag)
	}
	if stats.Uplink != 0 || stats.Downlink != 0 {
		t.Errorf("Expected zero stats for non-existent outbound, got Uplink=%d, Downlink=%d",
			stats.Uplink, stats.Downlink)
	}
}

func TestSingBoxAPI_GetOutboundStats_WithReset(t *testing.T) {
	api, cleanup := setupRealSingBoxWithUsers(t)
	defer cleanup()

	ctx := context.Background()

	stats, err := api.GetOutboundStats(ctx, "test-outbound", true)
	if err != nil {
		t.Fatalf("GetOutboundStats with reset failed: %v", err)
	}

	t.Logf("Outbound stats with reset: Uplink=%d, Downlink=%d",
		stats.Uplink, stats.Downlink)
}

func TestSingBoxAPI_GetOutboundStats_ContextCancellation(t *testing.T) {
	api, cleanup := setupRealSingBox(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := api.GetOutboundStats(ctx, "test-outbound", false)
	if err == nil {
		t.Error("Expected error with cancelled context")
	}
}

func TestSingBoxAPI_GetOutboundStats_QueryStatsError(t *testing.T) {
	mockAPI := createMockAPI()
	mockAPI.client = &MockStatsServiceClient{
		queryStatsFunc: func(ctx context.Context, req *api.QueryStatsRequest, opts ...grpc.CallOption) (*api.QueryStatsResponse, error) {
			return nil, fmt.Errorf("query error")
		},
	}

	_, err := mockAPI.GetOutboundStats(context.Background(), "test-outbound", false)
	if err == nil {
		t.Error("Expected query error")
	}
}

func TestSingBoxAPI_GetOutboundStats_OnlyUplink(t *testing.T) {
	mockAPI := createMockAPI()
	mockAPI.client = &MockStatsServiceClient{
		queryStatsFunc: func(ctx context.Context, req *api.QueryStatsRequest, opts ...grpc.CallOption) (*api.QueryStatsResponse, error) {
			stats := []*api.Stat{
				{Name: "outbound>>>test-outbound>>>traffic>>>uplink", Value: 1800},
			}
			return &api.QueryStatsResponse{Stat: stats}, nil
		},
	}

	result, err := mockAPI.GetOutboundStats(context.Background(), "test-outbound", false)
	if err != nil {
		t.Fatalf("GetOutboundStats failed: %v", err)
	}

	if result.Uplink != 1800 {
		t.Errorf("Expected uplink 1800, got %d", result.Uplink)
	}
	if result.Downlink != 0 {
		t.Errorf("Expected downlink 0, got %d", result.Downlink)
	}
}

func TestSingBoxAPI_GetOutboundStats_OnlyDownlink(t *testing.T) {
	mockAPI := createMockAPI()
	mockAPI.client = &MockStatsServiceClient{
		queryStatsFunc: func(ctx context.Context, req *api.QueryStatsRequest, opts ...grpc.CallOption) (*api.QueryStatsResponse, error) {
			stats := []*api.Stat{
				{Name: "outbound>>>test-outbound>>>traffic>>>downlink", Value: 2800},
			}
			return &api.QueryStatsResponse{Stat: stats}, nil
		},
	}

	result, err := mockAPI.GetOutboundStats(context.Background(), "test-outbound", false)
	if err != nil {
		t.Fatalf("GetOutboundStats failed: %v", err)
	}

	if result.Uplink != 0 {
		t.Errorf("Expected uplink 0, got %d", result.Uplink)
	}
	if result.Downlink != 2800 {
		t.Errorf("Expected downlink 2800, got %d", result.Downlink)
	}
}

func TestSingBoxAPI_GetOutboundStats_UnknownLink(t *testing.T) {
	mockAPI := createMockAPI()
	mockAPI.client = &MockStatsServiceClient{
		queryStatsFunc: func(ctx context.Context, req *api.QueryStatsRequest, opts ...grpc.CallOption) (*api.QueryStatsResponse, error) {
			stats := []*api.Stat{
				{Name: "outbound>>>test-outbound>>>traffic>>>unknown", Value: 3800},
			}
			return &api.QueryStatsResponse{Stat: stats}, nil
		},
	}

	result, err := mockAPI.GetOutboundStats(context.Background(), "test-outbound", false)
	if err != nil {
		t.Fatalf("GetOutboundStats failed: %v", err)
	}

	if result.Uplink != 0 {
		t.Errorf("Expected uplink 0, got %d", result.Uplink)
	}
	if result.Downlink != 0 {
		t.Errorf("Expected downlink 0, got %d", result.Downlink)
	}
}
