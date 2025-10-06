package node

import (
	"context"
	"log"
	"os/exec"

	proto "github.com/bookpanda/firecracker-runner-node/proto/node/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type Service interface {
	proto.NodeServiceServer
}

type serviceImpl struct {
	proto.UnimplementedNodeServiceServer
	manager *NodeManager
	log     *zap.Logger
}

func NewService(manager *NodeManager, log *zap.Logger) Service {
	return &serviceImpl{
		manager: manager,
		log:     log,
	}
}

func (s *serviceImpl) SendServerCommand(_ context.Context, req *proto.SendServerCommandNodeRequest) (*proto.SendServerCommandNodeResponse, error) {
	if err := s.manager.SendServerCommand(req.Command); err != nil {
		return nil, err
	}

	return &proto.SendServerCommandNodeResponse{}, nil
}

func (s *serviceImpl) SendClientCommand(req *proto.SendClientCommandNodeRequest, stream grpc.ServerStreamingServer[proto.SendClientCommandNodeResponse]) error {
	if err := s.manager.SendClientCommand(req.Command); err != nil {
		return err
	}

	response := &proto.SendClientCommandNodeResponse{Output: "Command finished executing"}
	if err := stream.Send(response); err != nil {
		return err
	}

	return nil
}

func (s *serviceImpl) StopSyscalls(_ context.Context, req *proto.StopSyscallsNodeRequest) (*proto.StopSyscallsNodeResponse, error) {
	if err := s.manager.StopSyscalls(); err != nil {
		return nil, err
	}

	return &proto.StopSyscallsNodeResponse{}, nil
}

func (s *serviceImpl) Cleanup(_ context.Context, req *proto.CleanupNodeRequest) (*proto.CleanupNodeResponse, error) {
	log.Printf("Cleaning up node...")
	cmd := exec.Command("sudo", "pkill", "-f", "iperf3")
	if err := cmd.Run(); err != nil {
		log.Printf("Warning: failed to kill iperf3 processes: %v", err)
	}

	cmd = exec.Command("sudo", "pkill", "-f", "sockperf")
	if err := cmd.Run(); err != nil {
		log.Printf("Warning: failed to kill sockperf processes: %v", err)
	}

	s.manager = NewManager(s.manager.config)

	return &proto.CleanupNodeResponse{}, nil
}
