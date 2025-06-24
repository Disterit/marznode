package api

import (
	"context"
	"marznode/api/pb"
	"marznode/internal/service"
	"marznode/pkg/backend/common"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type MarznodeHandler struct {
	marznode service.Marznode
	log      *zap.SugaredLogger
	pb.UnimplementedMarzServiceServer
	backends []common.VPNBackend
}

func NewMarznodeHandler(marznode service.Marznode, log *zap.SugaredLogger, backend ...common.VPNBackend) *MarznodeHandler {
	return &MarznodeHandler{
		marznode: marznode,
		log:      log,
		backends: backend,
	}
}

func (h *MarznodeHandler) SyncUsers(server grpc.ClientStreamingServer[pb.UserData, pb.Empty]) error {

	return nil
}

func (h *MarznodeHandler) RepopulateUsers(ctx context.Context, user *pb.UsersData) (*pb.Empty, error) {

	return nil, nil
}

func (h *MarznodeHandler) FetchBackends(ctx context.Context, empty *pb.Empty) (*pb.BackendsResponse, error) {

	return nil, nil
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
