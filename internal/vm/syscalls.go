package vm

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func (m *Manager) TrackSyscalls() error {
	tracePath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %v", err)
	}
	tracePath = filepath.Join(tracePath, "trace_syscalls.sh")

	for _, vm := range m.vms {
		pid, err := vm.Machine.PID()
		if err != nil {
			return fmt.Errorf("failed to get vm %s PID: %v", vm.IP, err)
		}

		command := fmt.Sprintf("sudo %s %d", tracePath, pid)
		logPath := filepath.Join(m.syscallsDir, fmt.Sprintf("vm-%s.log", vm.IP))
		if err := captureCommandOutput(m.traceCtx, vm.IP, command, logPath); err != nil {
			return fmt.Errorf("failed to track syscalls of vm %s: %v", vm.IP, err)
		}
	}
	return nil
}

func (m *Manager) StopSyscalls() error {
	m.cancelTrace()
	return nil
}

func captureCommandOutput(ctx context.Context, vmIP, command, logPath string) error {
	logFile, err := os.Create(logPath)
	if err != nil {
		return fmt.Errorf("failed to create log file %s: %v", logPath, err)
	}

	go func() {
		defer logFile.Close()

		var cmd *exec.Cmd
		// Split the command properly for exec
		args := strings.Fields(command)
		if len(args) == 0 {
			log.Printf("Empty trace command for %s", vmIP)
			return
		}
		cmd = exec.CommandContext(ctx, args[0], args[1:]...)

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			log.Printf("Failed to create stdout pipe for %s: %v", vmIP, err)
			return
		}

		stderr, err := cmd.StderrPipe()
		if err != nil {
			log.Printf("Failed to create stderr pipe for %s: %v", vmIP, err)
			return
		}

		if err := cmd.Start(); err != nil {
			log.Printf("Failed to start command on %s: %v", vmIP, err)
			return
		}

		// Track trace processes for proper cleanup
		// if isTrace {
		// 	e.traceProcsMux.Lock()
		// 	e.traceProcs = append(e.traceProcs, cmd)
		// 	e.traceProcsMux.Unlock()
		// }

		go func() {
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				logFile.WriteString(fmt.Sprintf("[STDOUT] %s\n", scanner.Text()))
			}
		}()

		go func() {
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				logFile.WriteString(fmt.Sprintf("[STDERR] %s\n", scanner.Text()))
			}
		}()

		// for trace: stop when traceCtx is canceled
		<-ctx.Done()
		cmd.Process.Kill() // kill ONLY server process
		cmd.Wait()         // wait for stdout/stderr to be closed
		log.Printf("Server on %s stopped, logs saved to %s", vmIP, logPath)
	}()

	return nil
}
