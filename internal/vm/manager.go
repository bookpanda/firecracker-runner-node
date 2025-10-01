package vm

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"sync"

	"github.com/bookpanda/firecracker-runner-node/internal/config"
)

type Manager struct {
	config      *config.Config
	vms         map[string]*SimplifiedVM
	wg          sync.WaitGroup
	syscallsDir string
	testDir     string
}

func NewManager(cfg *config.Config) *Manager {
	return &Manager{
		config:      cfg,
		vms:         make(map[string]*SimplifiedVM),
		syscallsDir: "./vm-syscalls",
		testDir:     "./vm-test",
	}
}

func (m *Manager) CreateVM(ctx context.Context, ip, kernelPath, rootfsPath string) (*SimplifiedVM, error) {
	vm, err := CreateVM(ctx, ip, kernelPath, rootfsPath, len(m.vms))
	if err != nil {
		return nil, err
	}

	if err := vm.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start VM %d: %v", len(m.vms), err)
	}
	log.Printf("VM %d started successfully. Socket: %s", len(m.vms), vm.SocketPath)

	m.vms[vm.IP] = vm
	return vm, nil
}

func (m *Manager) StopAllVMs(ctx context.Context) error {
	var wg sync.WaitGroup
	for _, vm := range m.vms {
		wg.Add(1)
		go func(vm *SimplifiedVM) {
			defer wg.Done()
			if err := vm.Stop(ctx); err != nil {
				log.Printf("Failed to stop VM %d: %v", vm.VMID, err)
			}
		}(vm)
	}
	wg.Wait()
	return nil
}

func (m *Manager) LogNetworkingInfo() {
	log.Printf("All %d VMs started successfully", len(m.vms))
	log.Println("VM networking setup:")
	for ip, vm := range m.vms {
		log.Printf("  VM %d: tap%d, MAC: AA:FC:00:00:00:%02X, IP: %s/24", ip, vm.VMID, vm.VMID+1, vm.IP)
	}
}

func (m *Manager) SendCommand(ctx context.Context, ip, command string) error {
	vm, ok := m.vms[ip]
	if !ok {
		return fmt.Errorf("vm %s not found", ip)
	}
	logPath := filepath.Join(m.testDir, fmt.Sprintf("vm-%s.log", vm.IP))

	if err := m.captureCommandOutputVsock(ctx, vm.VsockPath, vm.VsockCID, command, logPath, false); err != nil {
		return fmt.Errorf("failed to send command to vm %s: %v", vm.IP, err)
	}

	return nil
}
