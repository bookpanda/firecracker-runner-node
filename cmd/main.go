package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/bookpanda/firecracker-runner-node/internal/config"
	"github.com/bookpanda/firecracker-runner-node/internal/filesystem"
	"github.com/bookpanda/firecracker-runner-node/internal/network"
	"github.com/bookpanda/firecracker-runner-node/internal/node"
	"github.com/bookpanda/firecracker-runner-node/internal/vm"
	filesystemProto "github.com/bookpanda/firecracker-runner-node/proto/filesystem/v1"
	networkProto "github.com/bookpanda/firecracker-runner-node/proto/network/v1"
	nodeProto "github.com/bookpanda/firecracker-runner-node/proto/node/v1"
	vmProto "github.com/bookpanda/firecracker-runner-node/proto/vm/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

func main() {
	conf := config.ParseFlags()

	logger := zap.Must(zap.NewDevelopment())
	vmCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	vmManager := vm.NewManager(conf, vmCtx)
	vmSvc := vm.NewService(vmManager, logger.Named("vmSvc"))

	networkSvc := network.NewService(logger.Named("networkSvc"))
	filesystemSvc := filesystem.NewService(logger.Named("filesystemSvc"))

	nodeManager := node.NewManager(conf)
	nodeSvc := node.NewService(nodeManager, logger.Named("nodeSvc"))

	listener, err := net.Listen("tcp", fmt.Sprintf(":%v", conf.Port))
	if err != nil {
		panic(fmt.Sprintf("Failed to listen: %v", err))
	}

	grpcServer := grpc.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, health.NewServer())
	vmProto.RegisterVmServiceServer(grpcServer, vmSvc)
	networkProto.RegisterNetworkServiceServer(grpcServer, networkSvc)
	filesystemProto.RegisterFileSystemServiceServer(grpcServer, filesystemSvc)
	nodeProto.RegisterNodeServiceServer(grpcServer, nodeSvc)

	reflection.Register(grpcServer)
	go func() {
		logger.Sugar().Infof("Firecracker Runner starting at port %v", conf.Port)

		if err := grpcServer.Serve(listener); err != nil {
			logger.Fatal("Failed to start Firecracker Runner service", zap.Error(err))
		}
	}()

	wait := gracefulShutdown(context.Background(), 2*time.Second, logger, map[string]operation{
		"vm-manager": func(ctx context.Context) error {
			cancel() // cancel vmCtx to stop syscall tracking and other VM operations
			return vmManager.StopAllVMs()
		},
		"server": func(ctx context.Context) error {
			grpcServer.GracefulStop()
			return nil
		},
	})

	<-wait

	grpcServer.GracefulStop()
	logger.Info("Closing the listener")
	listener.Close()
	logger.Info("Firecracker Runner service has been shutdown gracefully")
}

type operation func(ctx context.Context) error

func gracefulShutdown(ctx context.Context, timeout time.Duration, log *zap.Logger, ops map[string]operation) <-chan struct{} {
	wait := make(chan struct{})
	go func() {
		s := make(chan os.Signal, 1)

		signal.Notify(s, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
		sig := <-s

		log.Named("graceful shutdown").Sugar().
			Infof("got signal \"%v\" shutting down service", sig)

		timeoutFunc := time.AfterFunc(timeout, func() {
			log.Named("graceful shutdown").Sugar().
				Errorf("timeout %v ms has been elapsed, force exit", timeout.Milliseconds())
			os.Exit(0)
		})

		defer timeoutFunc.Stop()

		var wg sync.WaitGroup

		for key, op := range ops {
			wg.Add(1)
			innerOp := op
			innerKey := key
			go func() {
				defer wg.Done()

				log.Named("graceful shutdown").Sugar().
					Infof("cleaning up: %v", innerKey)
				if err := innerOp(ctx); err != nil {
					log.Named("graceful shutdown").Sugar().
						Errorf("%v: clean up failed: %v", innerKey, err.Error())
					return
				}

				log.Named("graceful shutdown").Sugar().
					Infof("%v was shutdown gracefully", innerKey)
			}()
		}

		wg.Wait()
		close(wait)
	}()

	return wait
}
