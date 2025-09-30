package vm

import (
	"context"

	proto "github.com/bookpanda/firecracker-runner-node/proto/vm/v1"
	"go.uber.org/zap"
)

type Service interface {
	proto.VmServiceServer
}

type serviceImpl struct {
	proto.UnimplementedVmServiceServer
	log *zap.Logger
}

func NewService(log *zap.Logger) Service {
	return &serviceImpl{
		log: log,
	}
}

func (s *serviceImpl) Create(ctx context.Context, req *proto.CreateVmRequest) (*proto.CreateVmResponse, error) {
	return nil, nil
}

func (s *serviceImpl) SendCommand(ctx context.Context, req *proto.SendCommandVmRequest) (*proto.SendCommandVmResponse, error) {
	return nil, nil
}

func (s *serviceImpl) GetLogs(ctx context.Context, req *proto.GetLogsVmRequest) (*proto.GetLogsVmResponse, error) {
	return nil, nil
}
