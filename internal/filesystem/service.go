package filesystem

import (
	"context"
	"fmt"

	proto "github.com/bookpanda/firecracker-runner-node/proto/filesystem/v1"
	"go.uber.org/zap"
)

type Service interface {
	proto.FileSystemServiceServer
}

type serviceImpl struct {
	proto.UnimplementedFileSystemServiceServer
	log *zap.Logger
}

func NewService(log *zap.Logger) Service {
	return &serviceImpl{
		log: log,
	}
}

func (s *serviceImpl) Cleanup(ctx context.Context, req *proto.CleanupFileSystemRequest) (*proto.CleanupFileSystemResponse, error) {
	logDirs := []string{"./vm-test", "./vm-logs", "./vm-syscalls", "./node-logs"}
	for _, logDir := range logDirs {
		if err := GetEmptyLogDir(logDir); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %v", err)
		}
	}

	cleanFiles := []string{"vm-", "vsock-"}
	for _, cleanFile := range cleanFiles {
		if err := CleanFilesInDir("/tmp", cleanFile); err != nil {
			return nil, fmt.Errorf("failed to clean up %s files: %v", cleanFile, err)
		}
	}

	return &proto.CleanupFileSystemResponse{}, nil
}

func (s *serviceImpl) GetLogs(ctx context.Context, req *proto.GetLogsFileSystemRequest) (*proto.GetLogsFileSystemResponse, error) {
	return &proto.GetLogsFileSystemResponse{}, nil
}
