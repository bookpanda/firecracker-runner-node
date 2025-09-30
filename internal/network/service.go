package network

import (
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
