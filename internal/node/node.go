package node

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"sync"

	"github.com/bookpanda/firecracker-runner-node/internal/config"
)

type Node struct {
	config      *config.Config
	traceCtx    context.Context
	cancelTrace context.CancelFunc
	wg          sync.WaitGroup
	logsDir     string
}

func NewNode(cfg *config.Config) *Node {
	traceCtx, cancelTrace := context.WithCancel(context.Background())
	return &Node{
		config:      cfg,
		traceCtx:    traceCtx,
		cancelTrace: cancelTrace,
		wg:          sync.WaitGroup{},
		logsDir:     "./node-logs",
	}
}

func (n *Node) SendServerCommand(command string) error {
	testLogPath := filepath.Join(n.logsDir, "node-server.log")

	if err := captureCommandOutput(n.traceCtx, command, testLogPath); err != nil {
		log.Printf("failed to send command to node: %v", err)
		return fmt.Errorf("failed to send command to node: %v", err)
	}

	return nil
}

func (n *Node) SendClientCommand(command string) error {
	testLogPath := filepath.Join(n.logsDir, "node-client.log")

	if err := captureCommandOutput(n.traceCtx, command, testLogPath); err != nil {
		log.Printf("failed to send command to node: %v", err)
		return fmt.Errorf("failed to send command to node: %v", err)
	}

	n.wg.Wait()

	return nil
}
