package network

import (
	"fmt"
	"log"
	"os"
	"os/exec"
)

type Bridge struct {
	Name string
	IP   string
	Taps []string
}

func NewBridge(name, ip string) *Bridge {
	return &Bridge{
		Name: name,
		IP:   ip,
		Taps: make([]string, 0),
	}
}

func (b *Bridge) Setup() error {
	// create bridge
	cmd := exec.Command("sudo", "ip", "link", "add", "name", b.Name, "type", "bridge")
	if err := cmd.Run(); err != nil {
		log.Printf("Bridge creation: %v (might already exist)", err)
	}

	cmd = exec.Command("sudo", "ip", "link", "set", b.Name, "up")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to bring up bridge: %v", err)
	}

	return nil
}

func (b *Bridge) AddTapAndBringUp(tapName string) error {
	// create tap
	cmd := exec.Command("sudo", "ip", "tuntap", "add", "dev", tapName, "mode", "tap", "user", os.Getenv("USER"))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create tap interface %s: %v", tapName, err)
	}

	// add tap to bridge
	cmd = exec.Command("sudo", "ip", "link", "set", tapName, "master", b.Name)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add %s to bridge: %v", tapName, err)
	}

	// bring up tap
	cmd = exec.Command("sudo", "ip", "link", "set", tapName, "up")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to bring up %s: %v", tapName, err)
	}

	log.Printf("Added tap %s to bridge %s", tapName, b.Name)
	b.Taps = append(b.Taps, tapName)

	return nil
}
