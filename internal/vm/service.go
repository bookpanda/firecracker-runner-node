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
	manager *Manager
	log     *zap.Logger
}

func NewService(manager *Manager, log *zap.Logger) Service {
	return &serviceImpl{
		manager: manager,
		log:     log,
	}
}

func (s *serviceImpl) Create(ctx context.Context, req *proto.CreateVmRequest) (*proto.CreateVmResponse, error) {
	vm, err := s.manager.CreateVM(ctx, req.Ip, req.KernelPath, req.RootfsPath)
	if err != nil {
		return nil, err
	}
	return &proto.CreateVmResponse{Vm: &proto.Vm{Ip: vm.IP, KernelPath: vm.KernelPath, RootfsPath: vm.RootfsPath}}, nil
}

func (s *serviceImpl) SendCommand(ctx context.Context, req *proto.SendCommandVmRequest) (*proto.SendCommandVmResponse, error) {
	return nil, nil
}
