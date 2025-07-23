package app

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/tongshengw/nimbus/backend/sectionleader/internal/constants"
	"github.com/tongshengw/nimbus/backend/sectionleader/internal/shared"
)

const CniConfRootDir = "/etc/cni/conf.d"

type IPAM struct {
	Type   string  `json:"type"`
	Subnet string  `json:"subnet"`
	Routes []Route `json:"routes"`
}

type Route struct {
	Dst string `json:"dst"`
}

type Plugin struct {
	Type string `json:"type"`
	IPAM *IPAM  `json:"ipam,omitempty"`
}

type CNIConfig struct {
	CNIVersion string   `json:"cniVersion"`
	Name       string   `json:"name"`
	Plugins    []Plugin `json:"plugins"`
}

var nextSubnet net.IP = net.ParseIP(constants.CniFirstSubnetStr).To4()

// returns name of the config generated
func GenerateCniConfFile(id shared.MachineUUID) (string, error) {
	vmID := id.String()
	if nextSubnet.Equal(net.ParseIP(constants.CniLastSubnetStr).To4()) {
		return "", fmt.Errorf("ran out of subnet IDs")
	}

	subnet := nextSubnet.String() + "/30"

	if nextSubnet.To4() == nil {
		return "", fmt.Errorf("malformed next subnet ip")
	}
	subnetInt := binary.BigEndian.Uint32(nextSubnet.To4())
	binary.BigEndian.PutUint32(nextSubnet, subnetInt+4)

	// FIXME:
	logrus.Infof("HERE: %s", nextSubnet.String())

	config := CNIConfig{
		CNIVersion: "0.4.0",
		Name:       fmt.Sprintf("fcnet-%s", vmID),
		Plugins: []Plugin{
			{
				Type: "ptp",
				IPAM: &IPAM{
					Type:   "host-local",
					Subnet: subnet,
					Routes: []Route{
						{Dst: "0.0.0.0/0"},
					},
				},
			},
			{
				Type: "firewall",
			},
			{
				Type: "tc-redirect-tap",
			},
		},
	}

	jsonBytes, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", err
	}

	confPath := CniConfRootDir + "/fcnet-" + id.String() + ".conflist"
	err = os.MkdirAll(CniConfRootDir, 0755)
	if err != nil {
		return "", err
	}

	logrus.Infof("created CNI config, with subnet %s, path %s", subnet, confPath)
	return config.Name, os.WriteFile(confPath, jsonBytes, 0644)
}

// GenerateCniConfFileWithIP creates CNI configuration with a specific IP address
// Used for resurrecting existing VMs where we know the IP address
func GenerateCniConfFileWithIP(id shared.MachineUUID, ip shared.Ipv4) (string, error) {
	vmID := id.String()

	// Convert the provided IP to a subnet (assuming /30 as in the original function)
	parsedIP := net.ParseIP(ip.String())
	if parsedIP == nil {
		return "", fmt.Errorf("invalid IP address: %s", ip.String())
	}

	// Create a /30 subnet from the provided IP
	// For /30, we need to align to 4-byte boundaries
	ipInt := binary.BigEndian.Uint32(parsedIP.To4())
	subnetInt := ipInt & 0xFFFFFFFC // Clear last 2 bits for /30 subnet
	subnetIP := make(net.IP, 4)
	binary.BigEndian.PutUint32(subnetIP, subnetInt)

	subnet := subnetIP.String() + "/30"

	config := CNIConfig{
		CNIVersion: "0.4.0",
		Name:       fmt.Sprintf("fcnet-%s", vmID),
		Plugins: []Plugin{
			{
				Type: "ptp",
				IPAM: &IPAM{
					Type:   "host-local",
					Subnet: subnet,
					Routes: []Route{
						{Dst: "0.0.0.0/0"},
					},
				},
			},
			{
				Type: "firewall",
			},
			{
				Type: "tc-redirect-tap",
			},
		},
	}

	jsonBytes, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", err
	}

	confPath := CniConfRootDir + "/fcnet-" + id.String() + ".conflist"
	err = os.MkdirAll(CniConfRootDir, 0755)
	if err != nil {
		return "", err
	}

	logrus.Infof("created CNI config for resurrection, with subnet %s, path %s", subnet, confPath)
	return config.Name, os.WriteFile(confPath, jsonBytes, 0644)
}
