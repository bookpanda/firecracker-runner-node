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
	log *zap.Logger
}

func NewService(log *zap.Logger) Service {
	return &serviceImpl{
		log: log,
	}
}

func (s *serviceImpl) Setup(ctx context.Context, req *proto.SetupNetworkRequest) (*proto.SetupNetworkResponse, error) {
	taps, err := Setup(int(req.NumVMs))
	if err != nil {
		return nil, err
	}

	return &proto.SetupNetworkResponse{
		Taps: taps,
	}, nil
}

func (s *serviceImpl) Cleanup(ctx context.Context, req *proto.CleanupNetworkRequest) (*proto.CleanupNetworkResponse, error) {
	err := Cleanup(int(req.NumVMs))
	if err != nil {
		return nil, err
	}

	return &proto.CleanupNetworkResponse{}, nil
}
