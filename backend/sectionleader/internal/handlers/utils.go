package handlers

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/BurntSushi/toml"
	"github.com/tongshengw/nimbus/backend/sectionleader/internal/constants"
	"github.com/tongshengw/nimbus/backend/sectionleader/internal/models"
)

type proxyConfig struct {
	Name       string `toml:"name"`
	ConnType   string `toml:"type"`
	LocalIp    string `toml:"localIP"`
	LocalPort  int    `toml:"localPort"`
	RemotePort int    `toml:"remotePort"`
}

type frpcConfig struct {
	Proxies []proxyConfig `toml:"proxies"`
}

func CreateTomlFrpcConfig(data *models.MachineData) error {
	if data.RemotePort < constants.MinRemotePort || data.RemotePort > constants.MaxRemotePort {
		return fmt.Errorf("port requested outside allowed port range")
	}
	cfg := proxyConfig{
		Name:       data.Id.String(),
		ConnType:   "tcp",
		LocalIp:    data.LocalIp.String(),
		LocalPort:  22,
		RemotePort: data.RemotePort,
	}

	proxiesConfig := frpcConfig{
		Proxies: []proxyConfig{cfg},
	}

	err := os.MkdirAll(constants.FrpcConfigDir, 0755)
	if err != nil {
		return err
	}

	file, err := os.Create(constants.FrpcConfigDir + "/" + data.Id.String() + ".toml")
	if err != nil {
		return err
	}
	defer file.Close()

	err = toml.NewEncoder(file).Encode(proxiesConfig)
	if err != nil {
		return err
	}

	cmd := exec.Command("su", "tswu", "-c", constants.RefreshFrpcPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("reload frpc error, err: %v output: %s", err, output)
	}

	return nil
}
