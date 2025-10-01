package network

import (
	"context"

	proto "github.com/bookpanda/firecracker-runner-node/proto/network/v1"
	"go.uber.org/zap"
)

type Service interface {
	proto.NetworkServiceServer
}

type serviceImpl struct {
	proto.UnimplementedNetworkServiceServer
	bridge *Bridge
	log    *zap.Logger
}

func NewService(log *zap.Logger) Service {
	return &serviceImpl{
		bridge: NewBridge("", ""),
		log:    log,
	}
}

func (s *serviceImpl) Setup(ctx context.Context, req *proto.SetupNetworkRequest) (*proto.SetupNetworkResponse, error) {
	bridge, err := Setup(int(req.NumVMs), req.BridgeIP)
	if err != nil {
		return nil, err
	}

	s.bridge = bridge

	return &proto.SetupNetworkResponse{}, nil
}

func (s *serviceImpl) Cleanup(ctx context.Context, req *proto.CleanupNetworkRequest) (*proto.CleanupNetworkResponse, error) {
	err := Cleanup(int(req.NumVMs))
	if err != nil {
		return nil, err
	}

	return &proto.CleanupNetworkResponse{}, nil
}
