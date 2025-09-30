package network

import (
	"fmt"
	"log"
	"os/exec"
)

func Setup(numVMs int, subnet string, bridgeIP string) (*Bridge, error) {
	bridge := NewBridge("br0", bridgeIP, subnet)
	err := bridge.Setup()
	if err != nil {
		return nil, fmt.Errorf("failed to setup bridge: %v", err)
	}

	// create tap interfaces for each VM
	for i := 0; i < numVMs; i++ {
		err = bridge.AddTapAndBringUp(fmt.Sprintf("tap%d", i))
		if err != nil {
			return nil, fmt.Errorf("failed to add tap%d to bridge: %v", i, err)
		}
	}

	// configure bridge IP (only if not already configured)
	cmd := exec.Command("sh", "-c", fmt.Sprintf("ip addr show %s | grep -q inet", bridge.Name))
	if err := cmd.Run(); err != nil {
		// bridge doesn't have IP, add one
		cmd = exec.Command("sudo", "ip", "addr", "add", bridge.IP, "dev", bridge.Name)
		if err := cmd.Run(); err != nil {
			log.Printf("Failed to add IP to bridge (might already exist): %v", err)
		}
	}

	// set up iptables rules for forwarding
	setupIptables(bridge)

	log.Println("Networking setup completed successfully")
	return bridge, nil
}

func setupIptables(bridge *Bridge) {
	commands := [][]string{
		{"sudo", "sh", "-c", "echo 1 > /proc/sys/net/ipv4/ip_forward"},
		{"sudo", "iptables", "-I", "INPUT", "-i", bridge.Name, "-p", "udp", "-j", "ACCEPT"},
		{"sudo", "iptables", "-I", "INPUT", "-i", bridge.Name, "-p", "tcp", "-j", "ACCEPT"},
		{"sudo", "iptables", "-I", "FORWARD", "-i", bridge.Name, "-p", "udp", "-j", "ACCEPT"},
		{"sudo", "iptables", "-I", "FORWARD", "-i", bridge.Name, "-p", "tcp", "-j", "ACCEPT"},
		{"sudo", "iptables", "-I", "FORWARD", "1", "-i", bridge.Name, "-o", bridge.Name, "-j", "ACCEPT"},
	}

	for _, cmd := range commands {
		if err := exec.Command(cmd[0], cmd[1:]...).Run(); err != nil {
			log.Printf("iptables command failed (might already exist): %v", err)
		}
	}
}

func Cleanup(bridge *Bridge) error {
	log.Printf("Cleaning up networking...")
	err := bridge.Cleanup()
	if err != nil {
		return fmt.Errorf("failed to cleanup networking: %v", err)
	}
	log.Println("Networking cleanup completed")

	return nil
}
