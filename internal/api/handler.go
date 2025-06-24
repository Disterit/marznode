package api

import (
	"context"
	"encoding/json"
	"io"
	"marznode/api/pb"
	"marznode/internal/service"
	"marznode/pkg/backend/common"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type MarznodeHandler struct {
	marznode service.MarznodeMemory
	log      *zap.SugaredLogger
	pb.UnimplementedMarzServiceServer
	backends []common.VPNBackend
}

func NewMarznodeHandler(marznode service.MarznodeMemory, log *zap.SugaredLogger, backend ...common.VPNBackend) *MarznodeHandler {
	return &MarznodeHandler{
		marznode: marznode,
		log:      log,
		backends: backend,
	}
}

func (h *MarznodeHandler) SyncUsers(server grpc.ClientStreamingServer[pb.UserData, pb.Empty]) error {
	for {
		userData, err := server.Recv()
		if err == io.EOF {
			return server.SendAndClose(&pb.Empty{})
		}
		if err != nil {
			return err
		}

		h.log.Infof("Received user data: %v", userData)
	}
}

func (h *MarznodeHandler) RepopulateUsers(ctx context.Context, user *pb.UsersData) (*pb.Empty, error) {

	return nil, nil
}

func (h *MarznodeHandler) FetchBackends(ctx context.Context, empty *pb.Empty) (*pb.BackendsResponse, error) {
	var backends []*pb.Backend

	for i, backend := range h.backends {
		version, err := backend.Version()
		if err != nil {
			h.log.Warnf("Failed to get version for backend %d: %v", i, err)
			version = "unknown"
		}

		inbounds, err := backend.ListInbounds(ctx)
		if err != nil {
			h.log.Errorf("Failed to get inbounds for backend %d: %v", i, err)
			continue
		}

		var pbInbounds []*pb.Inbound
		for _, inbound := range inbounds {
			configJSON, err := json.Marshal(inbound.Config)
			if err != nil {
				h.log.Warnf("Failed to marshal config for inbound %s: %v", inbound.Tag, err)
				continue
			}

			configStr := string(configJSON)
			pbInbound := &pb.Inbound{
				Tag:    inbound.Tag,
				Config: &configStr,
			}
			pbInbounds = append(pbInbounds, pbInbound)
		}

		backendType := backend.BackendType()
		pbBackend := &pb.Backend{
			Name:     backendType,
			Type:     &backendType,
			Version:  &version,
			Inbounds: pbInbounds,
		}

		backends = append(backends, pbBackend)
	}

	return &pb.BackendsResponse{
		Backends: backends,
	}, nil
}

func (h *MarznodeHandler) FetchUsersStats(ctx context.Context, empty *pb.Empty) (*pb.UsersStats, error) {
	var allUserStats []*pb.UsersStats_UserStats

	for _, backend := range h.backends {
		stats, err := backend.GetUsages(ctx)
		if err != nil {
			return nil, err
		}
		if usageMap, ok := stats.(map[int64]int64); ok {
			for uid, usage := range usageMap {
				allUserStats = append(allUserStats, &pb.UsersStats_UserStats{
					Uid:   uint32(uid),
					Usage: uint64(usage),
				})
			}
		}
	}

	return &pb.UsersStats{
		UsersStats: allUserStats,
	}, nil
}

func (h *MarznodeHandler) FetchBackendConfig(ctx context.Context, backend *pb.Backend) (*pb.BackendConfig, error) {

	return nil, nil
}

func (h *MarznodeHandler) RestartBackend(ctx context.Context, request *pb.RestartBackendRequest) (*pb.Empty, error) {

	return nil, nil
}

func (h *MarznodeHandler) StreamBackendLogs(request *pb.BackendLogsRequest, client grpc.ServerStreamingServer[pb.LogLine]) error {

	return nil
}

func (h *MarznodeHandler) GetBackendStats(ctx context.Context, backend *pb.Backend) (*pb.BackendStats, error) {

	return nil, nil
}
