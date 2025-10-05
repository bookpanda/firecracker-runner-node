package network

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
)

func Setup(numVMs int, bridgeIP string) (*Bridge, error) {
	bridge := NewBridge("br0", bridgeIP)
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

	// configure bridge IP with subnet mask
	// Always ensure the bridge IP has /24 suffix
	bridgeIPWithMask := bridge.IP
	if !strings.Contains(bridgeIPWithMask, "/") {
		bridgeIPWithMask = bridgeIPWithMask + "/24"
	}

	cmd := exec.Command("sudo", "ip", "addr", "add", bridgeIPWithMask, "dev", bridge.Name)
	if err := cmd.Run(); err != nil {
		log.Printf("Failed to add IP to bridge (might already exist): %v", err)
	}

	// set up iptables rules for forwarding
	setupIptables(bridge)

	log.Println("Networking setup completed successfully")
	return bridge, nil
}

func setupIptables(bridge *Bridge) {
	// Extract subnet from bridge IP (e.g., "192.168.100.1" -> "192.168.100.0/24")
	bridgeSubnet := getBridgeSubnet(bridge.IP)

	commands := [][]string{
		{"sudo", "sh", "-c", "echo 1 > /proc/sys/net/ipv4/ip_forward"},
		{"sudo", "iptables", "-I", "INPUT", "-i", bridge.Name, "-p", "udp", "-j", "ACCEPT"},
		{"sudo", "iptables", "-I", "INPUT", "-i", bridge.Name, "-p", "tcp", "-j", "ACCEPT"},
		{"sudo", "iptables", "-I", "INPUT", "-i", bridge.Name, "-p", "icmp", "-j", "ACCEPT"},
		{"sudo", "iptables", "-I", "FORWARD", "-i", bridge.Name, "-p", "udp", "-j", "ACCEPT"},
		{"sudo", "iptables", "-I", "FORWARD", "-i", bridge.Name, "-p", "tcp", "-j", "ACCEPT"},
		{"sudo", "iptables", "-I", "FORWARD", "-i", bridge.Name, "-p", "icmp", "-j", "ACCEPT"},
		{"sudo", "iptables", "-I", "FORWARD", "-o", bridge.Name, "-p", "icmp", "-j", "ACCEPT"},
		{"sudo", "iptables", "-I", "FORWARD", "1", "-i", bridge.Name, "-o", bridge.Name, "-j", "ACCEPT"},
		// Enable masquerading for outgoing traffic from the bridge subnet
		{"sudo", "iptables", "-t", "nat", "-A", "POSTROUTING", "-s", bridgeSubnet, "!", "-d", bridgeSubnet, "-j", "MASQUERADE"},
	}

	for _, cmd := range commands {
		if err := exec.Command(cmd[0], cmd[1:]...).Run(); err != nil {
			log.Printf("iptables command failed (might already exist): %v", err)
		}
	}
}

func getBridgeSubnet(bridgeIP string) string {
	// Extract first 3 octets and add .0/24
	// e.g., "192.168.100.1" -> "192.168.100.0/24"
	parts := strings.Split(bridgeIP, ".")
	if len(parts) >= 3 {
		return fmt.Sprintf("%s.%s.%s.0/24", parts[0], parts[1], parts[2])
	}
	return "192.168.0.0/24" // fallback
}

func Cleanup(numVMs int) error {
	log.Printf("Cleaning up networking...")
	for i := 0; i < numVMs; i++ {
		tapName := fmt.Sprintf("tap%d", i)
		cmd := exec.Command("sudo", "ip", "link", "delete", tapName)
		if err := cmd.Run(); err != nil {
			log.Printf("failed to delete tap interface %s: %v, might already be deleted", tapName, err)
		}
	}

	cmd := exec.Command("sudo", "ip", "link", "delete", "br0")
	if err := cmd.Run(); err != nil {
		log.Printf("failed to delete bridge: %v, might already be deleted", err)
	}

	log.Println("Networking cleanup completed")
	return nil
}
