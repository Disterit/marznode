package singbox

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"slices"

	"github.com/highlight-apps/node-backend/backend/common/models"
	"github.com/highlight-apps/node-backend/storage"
	"github.com/highlight-apps/node-backend/utils"
)

type SingBoxConfig struct {
	Data          map[string]any            `json:"data"`
	ApiHost       string                    `json:"apiHost"`
	ApiPort       int                       `json:"apiPort"`
	Inbounds      []map[string]any          `json:"inbounds"`
	InboundsByTag map[string]map[string]any `json:"inboundsByTag"`
}

func NewSingBoxConfig(config string, apiHost string, apiPort int) (*SingBoxConfig, error) {
	var configData map[string]any

	if err := json.Unmarshal([]byte(config), &configData); err != nil {
		fileData, fileErr := os.ReadFile(config)
		if fileErr != nil {
			return nil, fmt.Errorf("failed to parse as JSON and failed to read as file: %v, %v", err, fileErr)
		}
		if err := json.Unmarshal(fileData, &configData); err != nil {
			return nil, fmt.Errorf("failed to parse file content as JSON: %v", err)
		}
	}

	c := &SingBoxConfig{
		Data:          configData,
		ApiHost:       apiHost,
		ApiPort:       apiPort,
		Inbounds:      make([]map[string]any, 0),
		InboundsByTag: make(map[string]map[string]any),
	}

	c.resolveInbounds()
	c.applyAPI()

	return c, nil
}

func (c *SingBoxConfig) applyAPI() {
	experimental, ok := c.Data["experimental"].(map[string]any)
	if !ok {
		experimental = make(map[string]any)
		c.Data["experimental"] = experimental
	}

	v2rayAPI, hasV2rayAPI := experimental["v2ray_api"].(map[string]any)
	if !hasV2rayAPI {
		v2rayAPI = make(map[string]any)
	}

	v2rayAPI["listen"] = c.ApiHost + ":" + strconv.Itoa(c.ApiPort)

	stats, hasStats := v2rayAPI["stats"].(map[string]any)
	if !hasStats {
		stats = make(map[string]any)
	}

	stats["enabled"] = true

	if _, hasUsers := stats["users"]; !hasUsers {
		stats["users"] = []string{}
	}

	v2rayAPI["stats"] = stats
	experimental["v2ray_api"] = v2rayAPI
}

func (c *SingBoxConfig) resolveInbounds() {
	inbounds, ok := c.Data["inbounds"].([]any)
	if !ok {
		return
	}

	supportedProtocols := map[string]bool{
		"shadowsocks": true,
		"vmess":       true,
		"trojan":      true,
		"vless":       true,
		"hysteria2":   true,
		"tuic":        true,
		"shadowtls":   true,
	}

	for _, item := range inbounds {
		inbound, ok := item.(map[string]any)
		if !ok {
			continue
		}

		inboundType, hasType := inbound["type"].(string)
		tag, hasTag := inbound["tag"].(string)

		if !hasType || !hasTag || !supportedProtocols[inboundType] {
			continue
		}

		settings := map[string]any{
			"tag":         tag,
			"protocol":    inboundType,
			"port":        inbound["listen_port"],
			"network":     nil,
			"tls":         "none",
			"sni":         []string{},
			"host":        []string{},
			"path":        nil,
			"header_type": nil,
			"flow":        nil,
		}

		if tls, hasTLS := inbound["tls"].(map[string]any); hasTLS {
			if enabled, ok := tls["enabled"].(bool); ok && enabled {
				settings["tls"] = "tls"
				if sni, ok := tls["server_name"].(string); ok && sni != "" {
					existingSni := settings["sni"].([]string)
					settings["sni"] = append(existingSni, sni)
				}

				if reality, hasReality := tls["reality"].(map[string]any); hasReality {
					if realityEnabled, ok := reality["enabled"].(bool); ok && realityEnabled {
						settings["tls"] = "reality"

						if pvk, ok := reality["private_key"].(string); ok {
							_, pubKey, err := utils.GetX25519(pvk)
							if err == nil {
								settings["pbk"] = pubKey
							}
						}

						if shortIDs, ok := reality["short_id"].([]any); ok && len(shortIDs) > 0 {
							if shortID, ok := shortIDs[0].(string); ok {
								settings["sid"] = shortID
							} else {
								settings["sid"] = ""
							}
						} else {
							settings["sid"] = ""
						}
					}
				}
			}
		}

		if transport, hasTransport := inbound["transport"].(map[string]any); hasTransport {
			if networkType, ok := transport["type"].(string); ok {
				settings["network"] = networkType

				switch networkType {
				case "ws":
					if path, ok := transport["path"].(string); ok {
						settings["path"] = path
					}
				case "http":
					if path, ok := transport["path"].(string); ok {
						settings["path"] = path
					}
					settings["network"] = "tcp"
					settings["header_type"] = "http"
					if host, ok := transport["host"].([]any); ok {
						hostStrs := make([]string, 0, len(host))
						for _, h := range host {
							if hStr, ok := h.(string); ok {
								hostStrs = append(hostStrs, hStr)
							}
						}
						settings["host"] = hostStrs
					}
				case "grpc":
					if serviceName, ok := transport["service_name"].(string); ok {
						settings["path"] = serviceName
					}
				case "httpupgrade":
					if path, ok := transport["path"].(string); ok {
						settings["path"] = path
					}
				}
			}
		}

		if inboundType == "shadowtls" {
			if version, ok := inbound["version"]; ok {
				settings["shadowtls_version"] = version
			}
		} else if inboundType == "hysteria2" {
			if _, hasObfs := inbound["obfs"]; hasObfs {
				if obfs, isMap := inbound["obfs"].(map[string]any); isMap {
					if obfsType, hasType := obfs["type"].(string); hasType {
						if password, hasPassword := obfs["password"].(string); hasPassword {
							settings["header_type"] = obfsType
							settings["path"] = password
						}
					}
				}
			}
		}

		c.Inbounds = append(c.Inbounds, settings)
		c.InboundsByTag[tag] = settings
	}
}

func (c *SingBoxConfig) AppendUser(user models.User, inbound models.Inbound) error {
	identifier := strconv.FormatInt(user.ID, 10) + "." + user.Username
	account, err := CreateAccount(inbound.Protocol, identifier, user.Key, nil)
	if err != nil {
		return fmt.Errorf("failed to create account: %v", err)
	}

	inbounds, ok := c.Data["inbounds"].([]any)
	if !ok {
		return fmt.Errorf("inbounds not found in config")
	}

	for i, item := range inbounds {
		inboundMap, ok := item.(map[string]any)
		if !ok {
			continue
		}

		if tag, ok := inboundMap["tag"].(string); ok && tag == inbound.Tag {
			users, hasUsers := inboundMap["users"].([]any)
			if !hasUsers {
				users = make([]any, 0)
			}

			users = append(users, account.ToDict())
			inboundMap["users"] = users

			experimental := c.Data["experimental"].(map[string]any)
			v2rayAPI := experimental["v2ray_api"].(map[string]any)
			stats := v2rayAPI["stats"].(map[string]any)

			var statsUsers []string
			if existingUsers, ok := stats["users"].([]any); ok {
				statsUsers = make([]string, 0, len(existingUsers))
				for _, u := range existingUsers {
					if userStr, ok := u.(string); ok {
						statsUsers = append(statsUsers, userStr)
					}
				}
			} else if existingUsers, ok := stats["users"].([]string); ok {
				statsUsers = existingUsers
			} else {
				statsUsers = make([]string, 0)
			}

			found := slices.Contains(statsUsers, identifier)

			if !found {
				statsUsers = append(statsUsers, identifier)
				stats["users"] = statsUsers
			}

			c.Data["inbounds"].([]any)[i] = inboundMap
			break
		}
	}

	return nil
}

func (c *SingBoxConfig) PopUser(user models.User, inbound models.Inbound) error {
	identifier := strconv.FormatInt(user.ID, 10) + "." + user.Username

	inbounds, ok := c.Data["inbounds"].([]any)
	if !ok {
		return fmt.Errorf("inbounds not found in config")
	}

	for i, item := range inbounds {
		inboundMap, ok := item.(map[string]any)
		if !ok {
			continue
		}

		if tag, ok := inboundMap["tag"].(string); ok && tag == inbound.Tag {
			users, hasUsers := inboundMap["users"].([]any)
			if !hasUsers {
				continue
			}

			var filteredUsers []any
			for _, userItem := range users {
				userMap, ok := userItem.(map[string]any)
				if !ok {
					continue
				}

				userName, hasName := userMap["name"].(string)
				username, hasUsername := userMap["username"].(string)

				if hasName && userName == identifier {
					continue
				}
				if hasUsername && username == identifier {
					continue
				}

				filteredUsers = append(filteredUsers, userItem)
			}

			inboundMap["users"] = filteredUsers
			c.Data["inbounds"].([]any)[i] = inboundMap
			break
		}
	}

	return nil
}

func (c *SingBoxConfig) RegisterInbounds(storage storage.BaseStorage) error {
	inbounds := c.ListInbounds()
	for _, inbound := range inbounds {
		if err := storage.RegisterInbound(inbound); err != nil {
			return fmt.Errorf("failed to register inbound %s: %v", inbound.Tag, err)
		}
	}
	return nil
}

func (c *SingBoxConfig) ListInbounds() []models.Inbound {
	var inbounds []models.Inbound
	for _, settings := range c.InboundsByTag {
		tag, _ := settings["tag"].(string)
		protocol, _ := settings["protocol"].(string)
		inbound := models.Inbound{
			Tag:      tag,
			Protocol: protocol,
			Config:   settings,
		}
		inbounds = append(inbounds, inbound)
	}
	return inbounds
}

func (c *SingBoxConfig) ToJSON() (string, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return "", fmt.Errorf("failed to encode to JSON: %v", err)
	}
	return string(data), nil
}
