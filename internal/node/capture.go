package node

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

func (n *NodeManager) trackSyscalls(pid int) error {
	log.Printf("NodeManager: Tracking syscalls for PID: %d", pid)
	tracePath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %v", err)
	}

	tracePath = filepath.Join(tracePath, "trace_syscalls.sh")
	command := fmt.Sprintf("sudo %s %d", tracePath, pid)
	logPath := filepath.Join(n.logsDir, fmt.Sprintf("node-syscalls-%d.log", pid))

	_, err = n.captureCommandOutput(n.traceCtx, command, logPath, false)
	if err != nil {
		return fmt.Errorf("failed to track syscalls of node: %v", err)
	}

	return nil
}

func (n *NodeManager) captureCommandOutput(ctx context.Context, command, logPath string, wait bool) (int, error) {
	logFile, err := os.Create(logPath)
	if err != nil {
		log.Printf("captureCommand: Failed to create log file %s: %v", logPath, err)
		return 0, fmt.Errorf("failed to create log file %s: %v", logPath, err)
	}

	// Split the command properly for exec
	args := strings.Fields(command)
	if len(args) == 0 {
		logFile.Close()
		return 0, fmt.Errorf("empty command")
	}
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		logFile.Close()
		return 0, fmt.Errorf("failed to create stdout pipe: %v", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		logFile.Close()
		return 0, fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		logFile.Close()
		return 0, fmt.Errorf("failed to start command: %v", err)
	}

	// Get PID immediately after start
	pid := cmd.Process.Pid
	log.Printf("Started command with PID: %d, wait=%v", pid, wait)

	if wait {
		n.wg.Add(1)
	}

	// Now handle I/O and waiting in goroutine
	go func() {
		if wait {
			defer n.wg.Done()
		}
		defer logFile.Close()

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

		if wait {
			err := cmd.Wait()
			if err != nil {
				log.Printf("Command (PID %d) completed with error: %v, output saved to %s", pid, err, logPath)
			} else {
				log.Printf("Command (PID %d) completed successfully, output saved to %s", pid, logPath)
			}
		} else {
			// for trace/server: stop when ctx is canceled
			log.Printf("Command (PID %d) running, waiting for context cancellation", pid)
			<-ctx.Done()
			log.Printf("Context cancelled, killing command (PID %d)", pid)
			cmd.Process.Kill()
			cmd.Wait()
			log.Printf("Command (PID %d) stopped, logs saved to %s", pid, logPath)
		}
	}()

	return pid, nil
}
