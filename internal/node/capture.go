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

func (n *Node) trackSyscalls(pid int) error {
	tracePath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %v", err)
	}

	tracePath = filepath.Join(tracePath, "trace_syscalls.sh")
	command := fmt.Sprintf("sudo %s %d", tracePath, pid)
	logPath := filepath.Join(n.logsDir, "node-syscalls.log")

	_, err = n.captureCommandOutput(n.traceCtx, command, logPath, false)
	if err != nil {
		return fmt.Errorf("failed to track syscalls of node: %v", err)
	}

	return nil
}

func (n *Node) captureCommandOutput(ctx context.Context, command, logPath string, wait bool) (int, error) {
	logFile, err := os.Create(logPath)
	if err != nil {
		return 0, fmt.Errorf("failed to create log file %s: %v", logPath, err)
	}

	if wait {
		n.wg.Add(1)
	}
	pidChan := make(chan int, 1)

	go func() {
		if wait {
			defer n.wg.Done()
		}
		defer logFile.Close()

		var cmd *exec.Cmd
		// Split the command properly for exec
		args := strings.Fields(command)
		if len(args) == 0 {
			log.Printf("Empty trace command for node")
			return
		}
		cmd = exec.CommandContext(ctx, args[0], args[1:]...)

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			log.Printf("Failed to create stdout pipe for node: %v", err)
			return
		}

		stderr, err := cmd.StderrPipe()
		if err != nil {
			log.Printf("Failed to create stderr pipe for node: %v", err)
			return
		}

		if err := cmd.Start(); err != nil {
			log.Printf("Failed to start command on node: %v", err)
			return
		}
		pid := cmd.Process.Pid
		pidChan <- pid
		log.Printf("Started command with PID: %d", pid)

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

		if wait {
			cmd.Wait()
			log.Printf("Command completed on node, output saved to %s", logPath)
		} else {
			// for trace/server: stop when ctx is canceled
			<-ctx.Done()
			cmd.Process.Kill() // kill ONLY server process
			cmd.Wait()         // wait for stdout/stderr to be closed
			log.Printf("Command on node stopped, logs saved to %s", logPath)
		}
	}()

	pid := <-pidChan
	if pid == 0 {
		return 0, fmt.Errorf("failed to get process PID")
	}

	return pid, nil
}
