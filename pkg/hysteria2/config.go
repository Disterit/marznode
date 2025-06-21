package hysteria2

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"marznode/internal/models"
	"strconv"
	"strings"
)

type HysteriaConfig struct {
	Config  map[string]interface{}
	Inbound models.Inbound
}

func (h *HysteriaConfig) NewHysteriaConfig(
	configYaml string,
	apiPort, statsPort int,
	statsSecret string,
) (*HysteriaConfig, error) {
	var loadedConfig map[string]interface{}
	err := yaml.Unmarshal([]byte(configYaml), &loadedConfig)
	if err != nil {
		return nil, fmt.Errorf("Error unmarshalling config: %v", err)
	}

	loadedConfig["auth"] = map[string]interface{}{
		"type": "http",
		"http": map[string]interface{}{
			"url": fmt.Sprintf("http://127.0.0.1:%d", apiPort),
		},
	}

	loadedConfig["trafficStats"] = map[string]interface{}{
		"listen": fmt.Sprintf("127.0.0.1:%d", statsPort),
		"secret": statsSecret,
	}

	port := 443
	if listen, ok := loadedConfig["listen"].(string); ok {
		parts := strings.Split(listen, ":")
		if len(parts) > 1 {
			if p, err := strconv.Atoi(parts[len(parts)-1]); err == nil {
				port = p
			}
		}
	}

	inboundConfig := map[string]interface{}{
		"tag":      "hysteria2",
		"protocol": "hysteria2",
		"port":     port,
		"tls":      "tls",
	}

	if obfs, ok := loadedConfig["obfs"].(map[string]interface{}); ok {
		if obsfType, ok := obfs["type"].(string); ok {
			if typedObfs, ok := obfs[obsfType].(map[string]interface{}); ok {
				if password, ok := typedObfs["password"].(string); ok {
					inboundConfig["path"] = password
					inboundConfig["header_type"] = obsfType
				}
			}
		}
	}

	return &HysteriaConfig{
		Config: loadedConfig,
		Inbound: models.Inbound{
			Tag:      "hysteria2",
			Protocol: "hysteria2",
			Config:   inboundConfig,
		},
	}, nil
}
