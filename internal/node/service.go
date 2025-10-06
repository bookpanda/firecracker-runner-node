package node

import (
	"context"

	proto "github.com/bookpanda/firecracker-runner-node/proto/node/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type Service interface {
	proto.NodeServiceServer
}

type serviceImpl struct {
	proto.UnimplementedNodeServiceServer
	node *NodeManager
	log  *zap.Logger
}

func NewService(node *NodeManager, log *zap.Logger) Service {
	return &serviceImpl{
		node: node,
		log:  log,
	}
}

func (s *serviceImpl) SendServerCommand(_ context.Context, req *proto.SendServerCommandNodeRequest) (*proto.SendServerCommandNodeResponse, error) {
	if err := s.node.SendServerCommand(req.Command); err != nil {
		return nil, err
	}

	return &proto.SendServerCommandNodeResponse{}, nil
}

func (s *serviceImpl) SendClientCommand(req *proto.SendClientCommandNodeRequest, stream grpc.ServerStreamingServer[proto.SendClientCommandNodeResponse]) error {
	if err := s.node.SendClientCommand(req.Command); err != nil {
		return err
	}

	response := &proto.SendClientCommandNodeResponse{Output: "Command finished executing"}
	if err := stream.Send(response); err != nil {
		return err
	}

	return nil
}

func (s *serviceImpl) StopSyscalls(_ context.Context, req *proto.StopSyscallsNodeRequest) (*proto.StopSyscallsNodeResponse, error) {
	if err := s.node.StopSyscalls(); err != nil {
		return nil, err
	}

	return &proto.StopSyscallsNodeResponse{}, nil
}
