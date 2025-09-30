package vm

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/firecracker-microvm/firecracker-go-sdk"
	"github.com/firecracker-microvm/firecracker-go-sdk/client/models"
)

type SimplifiedVM struct {
	Machine    *firecracker.Machine
	SocketPath string
	VsockPath  string
	VsockCID   uint32
	Stdout     chan string
	Stderr     chan string
	VMID       int
	TapName    string
	IP         string
}

func (v *SimplifiedVM) Start(ctx context.Context) error {
	if err := v.Machine.Start(ctx); err != nil {
		return fmt.Errorf("failed to start machine: %v", err)
	}

	go func() {
		for {
			select {
			case line := <-v.Stdout:
				if line != "" {
					v.Stdout <- line
				}
			case line := <-v.Stderr:
				if line != "" {
					v.Stderr <- line
				}
			case <-ctx.Done():
				close(v.Stdout)
				close(v.Stderr)
				return
			}
		}
	}()

	return nil
}

func (v *SimplifiedVM) Stop(ctx context.Context) error {
	if err := v.Machine.Shutdown(ctx); err != nil {
		log.Printf("Graceful shutdown failed for VM %d: %v", v.VMID, err)
	}

	// wait a bit for graceful shutdown to complete
	time.Sleep(2 * time.Second)

	if err := v.killFirecrackerProcess(); err != nil {
		log.Printf("Failed to kill Firecracker process for VM %d: %v", v.VMID, err)
	}

	// clean up socket file
	if err := os.Remove(v.SocketPath); err != nil && !os.IsNotExist(err) {
		log.Printf("failed to remove socket file: %v", err)
	}
	return nil
}

func (v *SimplifiedVM) killFirecrackerProcess() error {
	cmd := exec.Command("ps", "aux")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to list processes: %v", err)
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "firecracker") && strings.Contains(line, v.SocketPath) {
			// extract PID (second column in ps aux output)
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}

			pid, err := strconv.Atoi(fields[1])
			if err != nil {
				log.Printf("Failed to parse PID from line: %s", line)
				continue
			}

			log.Printf("Killing Firecracker process PID %d for VM %d", pid, v.VMID)

			// First try SIGTERM
			if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
				log.Printf("SIGTERM failed for PID %d: %v", pid, err)

				// If SIGTERM fails, try SIGKILL
				if err := syscall.Kill(pid, syscall.SIGKILL); err != nil {
					log.Printf("SIGKILL failed for PID %d: %v", pid, err)
					return fmt.Errorf("failed to kill process %d: %v", pid, err)
				}
			}

			// wait a moment for the process to die
			time.Sleep(500 * time.Millisecond)
		}
	}

	return nil
}

func CreateVM(ctx context.Context, kernelPath, rootfsPath string, vmIndex int) (*SimplifiedVM, error) {
	ip := fmt.Sprintf("192.168.100.%d", vmIndex+2)
	socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("vm-%s.sock", ip))
	vsockPath := filepath.Join(os.TempDir(), fmt.Sprintf("vsock-%s.sock", ip))
	cid := uint32(vmIndex + 3)

	stdout := make(chan string, 100)
	stderr := make(chan string, 100)

	macAddr := fmt.Sprintf("AA:FC:00:00:00:%02X", vmIndex+1)
	tapName := fmt.Sprintf("tap%d", vmIndex)

	logDir := "./vm-logs"

	cfg := firecracker.Config{
		SocketPath:      socketPath, // host-FC process communication
		KernelImagePath: kernelPath,
		KernelArgs:      "console=ttyS0 noapic reboot=k panic=1 pci=off rw",
		VsockDevices: []firecracker.VsockDevice{ // host-guest(vm) communication
			{
				ID:   fmt.Sprintf("vsock-%d", vmIndex),
				Path: vsockPath,
				CID:  cid,
			},
		},
		Drives: []models.Drive{
			{
				DriveID:      firecracker.String("1"),
				PathOnHost:   firecracker.String(rootfsPath),
				IsRootDevice: firecracker.Bool(true),
				IsReadOnly:   firecracker.Bool(true),
			},
		},
		NetworkInterfaces: []firecracker.NetworkInterface{
			{
				StaticConfiguration: &firecracker.StaticNetworkConfiguration{
					MacAddress:  macAddr,
					HostDevName: tapName,
					IPConfiguration: &firecracker.IPConfiguration{
						IPAddr: net.IPNet{
							IP:   net.ParseIP(ip),
							Mask: net.CIDRMask(24, 32),
						},
						Gateway:     net.ParseIP("192.168.100.254"),
						Nameservers: []string{"8.8.8.8", "8.8.4.4"},
					},
				},
			},
		},
		MachineCfg: models.MachineConfiguration{
			VcpuCount:  firecracker.Int64(1),
			MemSizeMib: firecracker.Int64(512),
		},
		ForwardSignals: []os.Signal{},
		LogLevel:       "Debug",
		LogPath:        filepath.Join(logDir, fmt.Sprintf("vm-%s.log", ip)),
		MetricsPath:    filepath.Join(logDir, fmt.Sprintf("vm-%s-metrics", ip)),
	}

	stdoutFile, err := os.Create(filepath.Join(logDir, fmt.Sprintf("vm-%s.stdout", ip)))
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout file: %v", err)
	}

	stderrFile, err := os.Create(filepath.Join(logDir, fmt.Sprintf("vm-%s.stderr", ip)))
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr file: %v", err)
	}

	cmd := firecracker.VMCommandBuilder{}.
		WithBin("firecracker").
		WithSocketPath(socketPath).
		WithStdin(os.Stdin).
		WithStdout(stdoutFile).
		WithStderr(stderrFile).
		Build(ctx)

	machine, err := firecracker.NewMachine(ctx, cfg, firecracker.WithProcessRunner(cmd))
	if err != nil {
		return nil, fmt.Errorf("failed to create machine: %v", err)
	}

	return &SimplifiedVM{
		Machine:    machine,
		SocketPath: socketPath,
		VsockPath:  vsockPath,
		VsockCID:   cid,
		Stdout:     stdout,
		Stderr:     stderr,
		VMID:       vmIndex,
		TapName:    tapName,
		IP:         ip,
	}, nil
}
