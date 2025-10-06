package node

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"sync"

	"github.com/bookpanda/firecracker-runner-node/internal/config"
)

type NodeManager struct {
	config      *config.Config
	traceCtx    context.Context
	cancelTrace context.CancelFunc
	wg          sync.WaitGroup
	logsDir     string
}

func NewManager(cfg *config.Config) *NodeManager {
	traceCtx, cancelTrace := context.WithCancel(context.Background())
	return &NodeManager{
		config:      cfg,
		traceCtx:    traceCtx,
		cancelTrace: cancelTrace,
		wg:          sync.WaitGroup{},
		logsDir:     "./node-logs",
	}
}

func (n *NodeManager) SendServerCommand(command string) error {
	testLogPath := filepath.Join(n.logsDir, "node-server.log")

	pid, err := n.captureCommandOutput(n.traceCtx, command, testLogPath, false)
	if err != nil {
		log.Printf("failed to send command to node: %v", err)
		return fmt.Errorf("failed to send command to node: %v", err)
	}

	if err := n.trackSyscalls(pid); err != nil {
		log.Printf("failed to track syscalls of node: %v", err)
		return fmt.Errorf("failed to track syscalls of node: %v", err)
	}

	return nil
}

func (n *NodeManager) SendClientCommand(command string) error {
	testLogPath := filepath.Join(n.logsDir, "node-client.log")

	pid, err := n.captureCommandOutput(n.traceCtx, command, testLogPath, true)
	if err != nil {
		log.Printf("failed to send command to node: %v", err)
		return fmt.Errorf("failed to send command to node: %v", err)
	}

	if err := n.trackSyscalls(pid); err != nil {
		log.Printf("failed to track syscalls of node: %v", err)
		return fmt.Errorf("failed to track syscalls of node: %v", err)
	}

	n.wg.Wait()

	return nil
}

func (n *NodeManager) StopSyscalls() error {
	n.cancelTrace()
	return nil
}
