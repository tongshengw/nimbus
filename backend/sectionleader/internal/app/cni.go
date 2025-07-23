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

func generateCniConfFileWithSubnet(id shared.MachineUUID, subnet string, logMessage string) (string, error) {
	vmID := id.String()

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

	logrus.Infof(logMessage, subnet, confPath)
	return config.Name, os.WriteFile(confPath, jsonBytes, 0644)
}

func GenerateCniConfFile(id shared.MachineUUID) (string, error) {
	if nextSubnet.Equal(net.ParseIP(constants.CniLastSubnetStr).To4()) {
		return "", fmt.Errorf("ran out of subnet IDs")
	}

	subnet := nextSubnet.String() + "/30"

	if nextSubnet.To4() == nil {
		return "", fmt.Errorf("malformed next subnet ip")
	}
	subnetInt := binary.BigEndian.Uint32(nextSubnet.To4())
	binary.BigEndian.PutUint32(nextSubnet, subnetInt+4)

	return generateCniConfFileWithSubnet(id, subnet, "created CNI config, with subnet %s, path %s")
}

func GenerateCniConfFileWithIP(id shared.MachineUUID, ip shared.Ipv4) (string, error) {
	parsedIP := net.ParseIP(ip.String())
	if parsedIP == nil {
		return "", fmt.Errorf("invalid IP address: %s", ip.String())
	}

	ipInt := binary.BigEndian.Uint32(parsedIP.To4())
	subnetInt := ipInt & 0xFFFFFFFC
	subnetIP := make(net.IP, 4)
	binary.BigEndian.PutUint32(subnetIP, subnetInt)

	subnet := subnetIP.String() + "/30"

	return generateCniConfFileWithSubnet(id, subnet, "created CNI config for resurrection, with subnet %s, path %s")
}
