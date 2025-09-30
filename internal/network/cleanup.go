package network

import (
	"fmt"
	"log"
	"os/exec"
	"time"
)

func CleanupExisting(numVMs int) error {
	log.Printf("Cleaning up any existing network interfaces...")

	for i := 0; i < numVMs+5; i++ {
		tapName := fmt.Sprintf("tap%d", i)
		cmd := exec.Command("sudo", "ip", "link", "delete", tapName)
		if err := cmd.Run(); err != nil {
			log.Printf("Tap interface %s cleanup: %v (might not exist)", tapName, err)
		}
	}

	cmd := exec.Command("sudo", "ip", "link", "delete", "br0")
	if err := cmd.Run(); err != nil {
		log.Printf("Bridge cleanup: %v (might not exist)", err)
	}

	time.Sleep(1 * time.Second)

	log.Println("Existing network cleanup completed")
	return nil
}

func Cleanup(numVMs int) error {
	log.Printf("Cleaning up networking for %d VMs...", numVMs)

	for i := 0; i < numVMs; i++ {
		tapName := fmt.Sprintf("tap%d", i)
		cmd := exec.Command("sudo", "ip", "link", "delete", tapName)
		if err := cmd.Run(); err != nil {
			log.Printf("Failed to delete tap interface %s: %v", tapName, err)
		}
	}

	cmd := exec.Command("sudo", "ip", "link", "delete", "br0")
	if err := cmd.Run(); err != nil {
		log.Printf("Failed to delete bridge: %v", err)
	}

	log.Println("Networking cleanup completed")
	return nil
}
