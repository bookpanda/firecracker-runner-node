package network

type NetworkConfig struct {
	BridgeName string
	BridgeIP   string
	Subnet     string
	NumVMs     int
}

func DefaultNetworkConfig(numVMs int) *NetworkConfig {
	return &NetworkConfig{
		BridgeName: "br0",
		BridgeIP:   "192.168.100.254/24",
		Subnet:     "192.168.100.0/24",
		NumVMs:     numVMs,
	}
}
