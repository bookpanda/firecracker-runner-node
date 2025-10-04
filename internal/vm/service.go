package vm

import (
	"context"
	"os/exec"

	proto "github.com/bookpanda/firecracker-runner-node/proto/vm/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
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

func (s *serviceImpl) Create(_ context.Context, req *proto.CreateVmRequest) (*proto.CreateVmResponse, error) {
	vm, err := s.manager.CreateVM(req.Ip, req.KernelPath, req.RootfsPath)
	if err != nil {
		return nil, err
	}
	return &proto.CreateVmResponse{Vm: &proto.Vm{Ip: vm.IP, KernelPath: vm.KernelPath, RootfsPath: vm.RootfsPath}}, nil
}

func (s *serviceImpl) SendCommand(_ context.Context, req *proto.SendCommandVmRequest) (*proto.SendCommandVmResponse, error) {
	if err := s.manager.SendCommand(req.Ip, req.Command, req.Wait); err != nil {
		return nil, err
	}

	return &proto.SendCommandVmResponse{}, nil
}

func (s *serviceImpl) SendClientCommand(req *proto.SendClientCommandVmRequest, stream grpc.ServerStreamingServer[proto.SendClientCommandVmResponse]) error {
	if err := s.manager.SendClientCommand(req.Ip, req.Command); err != nil {
		return err
	}

	response := &proto.SendClientCommandVmResponse{Output: "Command finished executing"}
	if err := stream.Send(response); err != nil {
		return err
	}

	return nil
}

func (s *serviceImpl) TrackSyscalls(_ context.Context, req *proto.TrackSyscallsVmRequest) (*proto.TrackSyscallsVmResponse, error) {
	if err := s.manager.TrackSyscalls(); err != nil {
		return nil, err
	}

	return &proto.TrackSyscallsVmResponse{}, nil
}

func (s *serviceImpl) StopSyscalls(_ context.Context, req *proto.StopSyscallsVmRequest) (*proto.StopSyscallsVmResponse, error) {
	if err := s.manager.StopSyscalls(); err != nil {
		return nil, err
	}

	return &proto.StopSyscallsVmResponse{}, nil
}

func (s *serviceImpl) Cleanup(_ context.Context, req *proto.CleanupVmRequest) (*proto.CleanupVmResponse, error) {
	cmd := exec.Command("sudo", "pkill", "-f", "firecracker")
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	s.manager = NewManager(s.manager.config, s.manager.vmCtx)

	return &proto.CleanupVmResponse{}, nil
}
