package singbox

import (
	"context"
	"fmt"
	"strings"

	"github.com/highlight-apps/node-backend/backend/singbox/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type GRPCClientFactory func(target string, opts ...grpc.DialOption) (*grpc.ClientConn, error)

var grpcClientFactory GRPCClientFactory = grpc.NewClient

type SysStatsResponse struct {
	NumGoroutine uint32 `json:"num_goroutine"`
	NumGC        uint32 `json:"num_gc"`
	Alloc        uint64 `json:"alloc"`
	TotalAlloc   uint64 `json:"total_alloc"`
	Sys          uint64 `json:"sys"`
	Mallocs      uint64 `json:"mallocs"`
	Frees        uint64 `json:"frees"`
	LiveObjects  uint64 `json:"live_objects"`
	PauseTotalNs uint64 `json:"pause_total_ns"`
	Uptime       uint32 `json:"uptime"`
}

type StatResponse struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Link  string `json:"link"`
	Value int64  `json:"value"`
}

type UserStatsResponse struct {
	Email    string `json:"email"`
	Uplink   int64  `json:"uplink"`
	Downlink int64  `json:"downlink"`
}

type InboundStatsResponse struct {
	Tag      string `json:"tag"`
	Uplink   int64  `json:"uplink"`
	Downlink int64  `json:"downlink"`
}

type OutboundStatsResponse struct {
	Tag      string `json:"tag"`
	Uplink   int64  `json:"uplink"`
	Downlink int64  `json:"downlink"`
}

type SingBoxAPIBase struct {
	address string
	port    int
	conn    *grpc.ClientConn
}

func NewSingBoxAPIBase(address string, port int) (*SingBoxAPIBase, error) {
	conn, err := grpcClientFactory(
		fmt.Sprintf("%s:%d", address, port),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	return &SingBoxAPIBase{
		address: address,
		port:    port,
		conn:    conn,
	}, nil
}

func (s *SingBoxAPIBase) Close() error {
	if s.conn != nil {
		return s.conn.Close()
	}
	return nil
}

type SingBoxAPI struct {
	*SingBoxAPIBase
	client api.StatsServiceClient
}

func NewSingBoxAPI(address string, port int) (*SingBoxAPI, error) {
	base, err := NewSingBoxAPIBase(address, port)
	if err != nil {
		return nil, err
	}

	return &SingBoxAPI{
		SingBoxAPIBase: base,
		client:         api.NewStatsServiceClient(base.conn),
	}, nil
}

func (s *SingBoxAPI) GetSysStats(ctx context.Context) (*SysStatsResponse, error) {
	response, err := s.client.GetSysStats(ctx, &api.SysStatsRequest{})
	if err != nil {
		return nil, err
	}

	return &SysStatsResponse{
		NumGoroutine: response.NumGoroutine,
		NumGC:        response.NumGC,
		Alloc:        response.Alloc,
		TotalAlloc:   response.TotalAlloc,
		Sys:          response.Sys,
		Mallocs:      response.Mallocs,
		Frees:        response.Frees,
		LiveObjects:  response.LiveObjects,
		PauseTotalNs: response.PauseTotalNs,
		Uptime:       response.Uptime,
	}, nil
}

func (s *SingBoxAPI) queryStats(ctx context.Context, pattern string, reset bool) ([]StatResponse, error) {
	response, err := s.client.QueryStats(ctx, &api.QueryStatsRequest{
		Pattern: pattern,
		Reset_:  reset,
	})
	if err != nil {
		return nil, err
	}

	var results []StatResponse
	for _, stat := range response.Stat {
		parts := strings.Split(stat.Name, ">>>")
		if len(parts) >= 4 {
			results = append(results, StatResponse{
				Name:  parts[1],
				Type:  parts[0],
				Link:  parts[3],
				Value: stat.Value,
			})
		}
	}
	return results, nil
}

func (s *SingBoxAPI) GetUsersStats(ctx context.Context, reset bool) ([]StatResponse, error) {
	return s.queryStats(ctx, "user>>>", reset)
}

func (s *SingBoxAPI) GetInboundsStats(ctx context.Context, reset bool) ([]StatResponse, error) {
	return s.queryStats(ctx, "inbound>>>", reset)
}

func (s *SingBoxAPI) GetOutboundsStats(ctx context.Context, reset bool) ([]StatResponse, error) {
	return s.queryStats(ctx, "outbound>>>", reset)
}

func (s *SingBoxAPI) GetUserStats(ctx context.Context, email string, reset bool) (*UserStatsResponse, error) {
	stats, err := s.queryStats(ctx, fmt.Sprintf("user>>>%s>>>", email), reset)
	if err != nil {
		return nil, err
	}

	var uplink, downlink int64
	for _, stat := range stats {
		switch stat.Link {
		case "uplink":
			uplink = stat.Value
		case "downlink":
			downlink = stat.Value
		}
	}

	return &UserStatsResponse{
		Email:    email,
		Uplink:   uplink,
		Downlink: downlink,
	}, nil
}

func (s *SingBoxAPI) GetInboundStats(ctx context.Context, tag string, reset bool) (*InboundStatsResponse, error) {
	stats, err := s.queryStats(ctx, fmt.Sprintf("inbound>>>%s>>>", tag), reset)
	if err != nil {
		return nil, err
	}

	var uplink, downlink int64
	for _, stat := range stats {
		switch stat.Link {
		case "uplink":
			uplink = stat.Value
		case "downlink":
			downlink = stat.Value
		}
	}

	return &InboundStatsResponse{
		Tag:      tag,
		Uplink:   uplink,
		Downlink: downlink,
	}, nil
}

func (s *SingBoxAPI) GetOutboundStats(ctx context.Context, tag string, reset bool) (*OutboundStatsResponse, error) {
	stats, err := s.queryStats(ctx, fmt.Sprintf("outbound>>>%s>>>", tag), reset)
	if err != nil {
		return nil, err
	}

	var uplink, downlink int64
	for _, stat := range stats {
		switch stat.Link {
		case "uplink":
			uplink = stat.Value
		case "downlink":
			downlink = stat.Value
		}
	}

	return &OutboundStatsResponse{
		Tag:      tag,
		Uplink:   uplink,
		Downlink: downlink,
	}, nil
}
