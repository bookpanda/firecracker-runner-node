package node

import (
	"context"
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
	node *Node
	log  *zap.Logger
}

func NewService(node *Node, log *zap.Logger) Service {
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

func (s *serviceImpl) Cleanup(_ context.Context, req *proto.CleanupNodeRequest) (*proto.CleanupNodeResponse, error) {
	cmd := exec.Command("sudo", "pkill", "-f", "firecracker")
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	s.node = NewNode(s.node.config)

	return &proto.CleanupNodeResponse{}, nil
}
