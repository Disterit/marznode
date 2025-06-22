package api

import (
	"context"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"marznode/api/pb"
	"marznode/internal/service"
)

type MarznodeHandler struct {
	marznode service.Marznode
	log      *zap.SugaredLogger
	pb.UnimplementedMarzServiceServer
}

func NewMarznodeHandler(marznode service.Marznode, log *zap.SugaredLogger) *MarznodeHandler {
	return &MarznodeHandler{
		marznode: marznode,
		log:      log,
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

	return nil, nil
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
