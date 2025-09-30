package network

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	proto "github.com/bookpanda/firecracker-runner-node/proto/network/v1"
)

func Setup(numVMs int) ([]*proto.Tap, error) {
	var taps []*proto.Tap

	config := DefaultNetworkConfig(numVMs)
	log.Printf("Setting up networking for %d VMs...", numVMs)

	// create bridge
	cmd := exec.Command("sudo", "ip", "link", "add", "name", config.BridgeName, "type", "bridge")
	if err := cmd.Run(); err != nil {
		log.Printf("Bridge creation: %v (might already exist)", err)
	}

	cmd = exec.Command("sudo", "ip", "link", "set", config.BridgeName, "up")
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to bring up bridge: %v", err)
	}

	// create tap interfaces for each VM
	for i := 0; i < numVMs; i++ {
		tapName := fmt.Sprintf("tap%d", i)

		// create tap
		cmd = exec.Command("sudo", "ip", "tuntap", "add", "dev", tapName, "mode", "tap", "user", os.Getenv("USER"))
		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("failed to create tap interface %s: %v", tapName, err)
		}

		// add tap to bridge
		cmd = exec.Command("sudo", "ip", "link", "set", tapName, "master", config.BridgeName)
		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("failed to add %s to bridge: %v", tapName, err)
		}

		// bring up tap
		cmd = exec.Command("sudo", "ip", "link", "set", tapName, "up")
		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("failed to bring up %s: %v", tapName, err)
		}

		log.Printf("Created and configured tap interface: %s", tapName)
		taps = append(taps, &proto.Tap{
			TapName: tapName,
			Ip:      fmt.Sprintf("192.168.100.%d", i),
			NodeIp:  fmt.Sprintf("192.168.100.%d", i),
			Mac:     fmt.Sprintf("02:00:00:00:00:0%d", i),
		})
	}

	// configure bridge IP (only if not already configured)
	cmd = exec.Command("sh", "-c", fmt.Sprintf("ip addr show %s | grep -q inet", config.BridgeName))
	if err := cmd.Run(); err != nil {
		// bridge doesn't have IP, add one
		cmd = exec.Command("sudo", "ip", "addr", "add", config.BridgeIP, "dev", config.BridgeName)
		if err := cmd.Run(); err != nil {
			log.Printf("Failed to add IP to bridge (might already exist): %v", err)
		}
	}

	// set up iptables rules for forwarding
	setupIptables(config)

	log.Println("Networking setup completed successfully")
	return taps, nil
}

func setupIptables(config *NetworkConfig) {
	commands := [][]string{
		{"sudo", "sh", "-c", "echo 1 > /proc/sys/net/ipv4/ip_forward"},
		{"sudo", "iptables", "-I", "INPUT", "-i", config.BridgeName, "-p", "udp", "-j", "ACCEPT"},
		{"sudo", "iptables", "-I", "INPUT", "-i", config.BridgeName, "-p", "tcp", "-j", "ACCEPT"},
		{"sudo", "iptables", "-I", "FORWARD", "-i", config.BridgeName, "-p", "udp", "-j", "ACCEPT"},
		{"sudo", "iptables", "-I", "FORWARD", "-i", config.BridgeName, "-p", "tcp", "-j", "ACCEPT"},
		{"sudo", "iptables", "-I", "FORWARD", "1", "-i", config.BridgeName, "-o", config.BridgeName, "-j", "ACCEPT"},
	}

	for _, cmd := range commands {
		if err := exec.Command(cmd[0], cmd[1:]...).Run(); err != nil {
			log.Printf("iptables command failed (might already exist): %v", err)
		}
	}
}
