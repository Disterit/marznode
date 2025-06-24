package singbox

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/highlight-apps/node-backend/backend/common/models"
)

func TestNewSingBoxConfig(t *testing.T) {
	t.Run("with inbounds", func(t *testing.T) {
		config := `{
		"inbounds": [
			{
				"type": "shadowsocks",
				"tag": "ss-in",
				"listen_port": 1080,
				"users": []
			},
			{
				"type": "vmess",
				"tag": "vmess-in",
				"listen_port": 1081,
				"users": [],
				"tls": {
					"enabled": true,
					"server_name": "example.com"
				},
				"transport": {
					"type": "ws",
					"path": "/path"
				}
			}
		]
	}`

		singboxConfig, err := NewSingBoxConfig(config, "127.0.0.1", 8080)
		if err != nil {
			t.Fatalf("Failed to create SingBoxConfig: %v", err)
		}

		if len(singboxConfig.Inbounds) != 2 {
			t.Errorf("Expected 2 inbounds, got %d", len(singboxConfig.Inbounds))
		}

		if len(singboxConfig.InboundsByTag) != 2 {
			t.Errorf("Expected 2 inbounds by tag, got %d", len(singboxConfig.InboundsByTag))
		}

		ssInbound, exists := singboxConfig.InboundsByTag["ss-in"]
		if !exists {
			t.Error("ss-in inbound not found in inboundsByTag")
		} else {
			if ssInbound["protocol"] != "shadowsocks" {
				t.Errorf("Expected shadowsocks protocol, got %v", ssInbound["protocol"])
			}
			if ssInbound["port"] != float64(1080) {
				t.Errorf("Expected port 1080, got %v", ssInbound["port"])
			}
		}

		vmessInbound, exists := singboxConfig.InboundsByTag["vmess-in"]
		if !exists {
			t.Error("vmess-in inbound not found in inboundsByTag")
		} else {
			if vmessInbound["protocol"] != "vmess" {
				t.Errorf("Expected vmess protocol, got %v", vmessInbound["protocol"])
			}
			if vmessInbound["tls"] != "tls" {
				t.Errorf("Expected tls enabled, got %v", vmessInbound["tls"])
			}
			if vmessInbound["network"] != "ws" {
				t.Errorf("Expected ws network, got %v", vmessInbound["network"])
			}
			if vmessInbound["path"] != "/path" {
				t.Errorf("Expected path /path, got %v", vmessInbound["path"])
			}
		}
	})

	t.Run("without inbounds", func(t *testing.T) {
		config := `{
		"outbounds": [
			{
				"type": "direct",
				"tag": "direct"
			}
		]
	}`

		singboxConfig, err := NewSingBoxConfig(config, "127.0.0.1", 8080)
		if err != nil {
			t.Fatalf("Failed to create SingBoxConfig: %v", err)
		}

		if len(singboxConfig.Inbounds) != 0 {
			t.Errorf("Expected 0 inbounds, got %d", len(singboxConfig.Inbounds))
		}

		if len(singboxConfig.InboundsByTag) != 0 {
			t.Errorf("Expected 0 inbounds by tag, got %d", len(singboxConfig.InboundsByTag))
		}
	})

	t.Run("edge cases", func(t *testing.T) {
		testCases := []struct {
			name        string
			config      string
			expectError bool
		}{
			{
				name:        "invalid JSON string",
				config:      `{"invalid": json}`,
				expectError: true,
			},
			{
				name:        "non-existent file",
				config:      "/non/existent/file.json",
				expectError: true,
			},
			{
				name:        "empty JSON object",
				config:      "{}",
				expectError: false,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := NewSingBoxConfig(tc.config, "127.0.0.1", 8080)
				if tc.expectError && err == nil {
					t.Error("Expected error, got nil")
				}
				if !tc.expectError && err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			})
		}
	})

	t.Run("valid file path", func(t *testing.T) {
		tmpFile := "/tmp/valid_config.json"
		validConfig := `{
		"inbounds": [
			{
				"type": "shadowsocks",
				"tag": "ss-in",
					"listen_port": 1080
			}
		]
	}`
		err := os.WriteFile(tmpFile, []byte(validConfig), 0644)
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile)

		config, err := NewSingBoxConfig(tmpFile, "127.0.0.1", 8080)
		if err != nil {
			t.Fatalf("Expected success for valid file, got error: %v", err)
		}

		if len(config.InboundsByTag) != 1 {
			t.Errorf("Expected 1 inbound, got %d", len(config.InboundsByTag))
		}
	})
}

func TestApplyAPI(t *testing.T) {
	t.Run("create experimental section from scratch", func(t *testing.T) {
		config := &SingBoxConfig{
			Data:    make(map[string]any),
			ApiHost: "127.0.0.1",
			ApiPort: 8080,
		}

		config.applyAPI()

		experimental, ok := config.Data["experimental"].(map[string]any)
		if !ok {
			t.Fatal("experimental section not created")
		}

		v2rayAPI, ok := experimental["v2ray_api"].(map[string]any)
		if !ok {
			t.Fatal("v2ray_api section not created")
		}

		if v2rayAPI["listen"] != "127.0.0.1:8080" {
			t.Errorf("Expected listen '127.0.0.1:8080', got %v", v2rayAPI["listen"])
		}

		stats, ok := v2rayAPI["stats"].(map[string]any)
		if !ok {
			t.Fatal("stats section not created")
		}

		if stats["enabled"] != true {
			t.Errorf("Expected stats enabled to be true, got %v", stats["enabled"])
		}

		users, ok := stats["users"].([]string)
		if !ok {
			t.Fatal("users array not created")
		}

		if len(users) != 0 {
			t.Errorf("Expected empty users array, got %d users", len(users))
		}
	})

	t.Run("preserve existing experimental config", func(t *testing.T) {
		config := &SingBoxConfig{
			Data: map[string]any{
				"experimental": map[string]any{
					"cache_file": map[string]any{
						"enabled": true,
						"path":    "/tmp/cache.db",
					},
				},
			},
			ApiHost: "192.168.1.100",
			ApiPort: 9090,
		}

		config.applyAPI()

		experimental, ok := config.Data["experimental"].(map[string]any)
		if !ok {
			t.Fatal("experimental section lost")
		}

		cacheFile, ok := experimental["cache_file"].(map[string]any)
		if !ok {
			t.Error("existing cache_file config lost")
		} else {
			if cacheFile["enabled"] != true {
				t.Error("cache_file enabled setting lost")
			}
			if cacheFile["path"] != "/tmp/cache.db" {
				t.Error("cache_file path setting lost")
			}
		}

		v2rayAPI, ok := experimental["v2ray_api"].(map[string]any)
		if !ok {
			t.Fatal("v2ray_api section not created")
		}

		if v2rayAPI["listen"] != "192.168.1.100:9090" {
			t.Errorf("Expected listen '192.168.1.100:9090', got %v", v2rayAPI["listen"])
		}
	})

	t.Run("preserve existing v2ray_api config", func(t *testing.T) {
		config := &SingBoxConfig{
			Data: map[string]any{
				"experimental": map[string]any{
					"v2ray_api": map[string]any{
						"tag": "api",
					},
				},
			},
			ApiHost: "127.0.0.1",
			ApiPort: 8080,
		}

		config.applyAPI()

		experimental := config.Data["experimental"].(map[string]any)
		v2rayAPI := experimental["v2ray_api"].(map[string]any)

		if v2rayAPI["tag"] != "api" {
			t.Error("existing v2ray_api tag lost")
		}

		if v2rayAPI["listen"] != "127.0.0.1:8080" {
			t.Errorf("Expected listen '127.0.0.1:8080', got %v", v2rayAPI["listen"])
		}
	})

	t.Run("preserve existing stats config", func(t *testing.T) {
		config := &SingBoxConfig{
			Data: map[string]any{
				"experimental": map[string]any{
					"v2ray_api": map[string]any{
						"stats": map[string]any{
							"users":    []string{"existing.user"},
							"inbounds": []string{"existing-inbound"},
						},
					},
				},
			},
			ApiHost: "127.0.0.1",
			ApiPort: 8080,
		}

		config.applyAPI()

		experimental := config.Data["experimental"].(map[string]any)
		v2rayAPI := experimental["v2ray_api"].(map[string]any)
		stats := v2rayAPI["stats"].(map[string]any)

		if stats["enabled"] != true {
			t.Error("stats enabled not set")
		}

		users, ok := stats["users"].([]string)
		if !ok {
			t.Fatal("users not preserved as string array")
		}

		if len(users) != 1 || users[0] != "existing.user" {
			t.Errorf("Expected existing users preserved, got %v", users)
		}

		inbounds, ok := stats["inbounds"].([]string)
		if !ok {
			t.Error("existing inbounds config lost")
		} else {
			if len(inbounds) != 1 || inbounds[0] != "existing-inbound" {
				t.Error("existing inbounds values lost")
			}
		}
	})
}

func TestResolveInbounds(t *testing.T) {
	t.Run("TLS and Reality parsing", func(t *testing.T) {
		config := `{
		"inbounds": [
			{
				"type": "vless",
				"tag": "vless-reality",
				"listen_port": 443,
				"users": [],
				"tls": {
					"enabled": true,
					"server_name": "example.com",
					"reality": {
						"enabled": true,
						"private_key": "YKQWZBUWbhZhQQJJGgZQQQQQQQQQQQQQQQQQQQQQQQQ=",
						"short_id": ["abc123"]
					}
				}
			},
			{
				"type": "vless",
				"tag": "vless-reality-empty-sid",
				"listen_port": 444,
				"users": [],
				"tls": {
					"enabled": true,
					"reality": {
						"enabled": true,
						"private_key": "YKQWZBUWbhZhQQJJGgZQQQQQQQQQQQQQQQQQQQQQQQQ="
					}
				}
			},
			{
				"type": "vless",
				"tag": "vless-tls-only",
				"listen_port": 445,
				"users": [],
				"tls": {
					"enabled": true,
					"server_name": "test.com"
				}
			}
		]
	}`

		singboxConfig, err := NewSingBoxConfig(config, "127.0.0.1", 8080)
		if err != nil {
			t.Fatalf("Failed to create SingBoxConfig: %v", err)
		}

		realityInbound := singboxConfig.InboundsByTag["vless-reality"]
		if realityInbound["tls"] != "reality" {
			t.Errorf("Expected reality TLS, got %v", realityInbound["tls"])
		}
		if realityInbound["sid"] != "abc123" {
			t.Errorf("Expected sid 'abc123', got %v", realityInbound["sid"])
		}
		if pbk, ok := realityInbound["pbk"].(string); !ok || pbk == "" {
			t.Errorf("Expected non-empty public key, got %v", pbk)
		}
		if sni, ok := realityInbound["sni"].([]string); !ok || len(sni) != 1 || sni[0] != "example.com" {
			t.Errorf("Expected SNI ['example.com'], got %v", sni)
		}

		emptySidInbound := singboxConfig.InboundsByTag["vless-reality-empty-sid"]
		if emptySidInbound["tls"] != "reality" {
			t.Errorf("Expected reality TLS, got %v", emptySidInbound["tls"])
		}
		if emptySidInbound["sid"] != "" {
			t.Errorf("Expected empty sid, got %v", emptySidInbound["sid"])
		}

		tlsOnlyInbound := singboxConfig.InboundsByTag["vless-tls-only"]
		if tlsOnlyInbound["tls"] != "tls" {
			t.Errorf("Expected tls, got %v", tlsOnlyInbound["tls"])
		}
		if sni, ok := tlsOnlyInbound["sni"].([]string); !ok || len(sni) != 1 || sni[0] != "test.com" {
			t.Errorf("Expected SNI ['test.com'], got %v", sni)
		}
	})

	t.Run("Hysteria2 obfs handling", func(t *testing.T) {
		config := `{
		"inbounds": [
			{
				"type": "hysteria2",
				"tag": "hysteria2-full",
				"listen_port": 443,
				"users": [],
				"obfs": {
					"type": "salamander",
					"password": "secret123"
				}
			},
			{
				"type": "hysteria2",
				"tag": "hysteria2-missing-password",
				"listen_port": 444,
				"users": [],
				"obfs": {
					"type": "salamander"
				}
			},
			{
				"type": "hysteria2",
				"tag": "hysteria2-no-obfs",
				"listen_port": 446,
				"users": []
			}
		]
	}`

		singboxConfig, err := NewSingBoxConfig(config, "127.0.0.1", 8080)
		if err != nil {
			t.Fatalf("Failed to create SingBoxConfig: %v", err)
		}

		fullObfs := singboxConfig.InboundsByTag["hysteria2-full"]
		if fullObfs["header_type"] != "salamander" {
			t.Errorf("Expected header_type 'salamander', got %v", fullObfs["header_type"])
		}
		if fullObfs["path"] != "secret123" {
			t.Errorf("Expected path 'secret123', got %v", fullObfs["path"])
		}

		missingPassword := singboxConfig.InboundsByTag["hysteria2-missing-password"]
		if missingPassword["header_type"] != nil {
			t.Errorf("Expected nil header_type when password missing, got %v", missingPassword["header_type"])
		}

		noObfs := singboxConfig.InboundsByTag["hysteria2-no-obfs"]
		if noObfs["header_type"] != nil {
			t.Errorf("Expected nil header_type when obfs missing, got %v", noObfs["header_type"])
		}
	})

	t.Run("transport types", func(t *testing.T) {
		config := `{
			"inbounds": [
				{
					"type": "vmess",
					"tag": "grpc-in",
					"listen_port": 1080,
					"transport": {
						"type": "grpc",
						"service_name": "VmessService"
					}
				},
				{
					"type": "vmess",
					"tag": "httpupgrade-in",
					"listen_port": 1081,
					"transport": {
						"type": "httpupgrade",
						"path": "/upgrade"
					}
				},
				{
					"type": "vmess",
					"tag": "host-array",
					"listen_port": 1082,
					"transport": {
						"type": "http",
						"path": "/http",
						"host": ["host1.com", "host2.com"]
					}
				}
			]
		}`

		singboxConfig, err := NewSingBoxConfig(config, "127.0.0.1", 8080)
		if err != nil {
			t.Fatalf("Failed to create SingBoxConfig: %v", err)
		}

		grpcInbound := singboxConfig.InboundsByTag["grpc-in"]
		if grpcInbound["network"] != "grpc" {
			t.Errorf("Expected grpc network, got %v", grpcInbound["network"])
		}
		if grpcInbound["path"] != "VmessService" {
			t.Errorf("Expected service name as path, got %v", grpcInbound["path"])
		}

		httpUpgradeInbound := singboxConfig.InboundsByTag["httpupgrade-in"]
		if httpUpgradeInbound["network"] != "httpupgrade" {
			t.Errorf("Expected httpupgrade network, got %v", httpUpgradeInbound["network"])
		}

		hostArray := singboxConfig.InboundsByTag["host-array"]
		if hostArray["network"] != "tcp" {
			t.Errorf("Expected tcp network, got %v", hostArray["network"])
		}
		if hostArray["header_type"] != "http" {
			t.Errorf("Expected http header_type, got %v", hostArray["header_type"])
		}
		host := hostArray["host"].([]string)
		if len(host) != 2 || host[0] != "host1.com" || host[1] != "host2.com" {
			t.Errorf("Expected host array [host1.com, host2.com], got %v", host)
		}
	})

	t.Run("edge cases", func(t *testing.T) {
		config := `{
			"inbounds": [
				{
					"type": "unsupported",
					"tag": "unsupported-in",
					"listen_port": 1080
				},
				{
					"type": "shadowsocks",
					"listen_port": 1080
				},
				{
					"tag": "test-in",
					"listen_port": 1080
				},
				{
					"type": "vless",
					"tag": "vless-non-string-sid",
					"listen_port": 443,
					"users": [],
					"tls": {
						"enabled": true,
						"reality": {
							"enabled": true,
							"private_key": "YKQWZBUWbhZhQQJJGgZQQQQQQQQQQQQQQQQQQQQQQQQ=",
							"short_id": [123]
						}
					}
				}
			]
		}`

		singboxConfig, err := NewSingBoxConfig(config, "127.0.0.1", 8080)
		if err != nil {
			t.Fatalf("Failed to create SingBoxConfig: %v", err)
		}

		if len(singboxConfig.InboundsByTag) != 1 {
			t.Errorf("Expected 1 valid inbound, got %d", len(singboxConfig.InboundsByTag))
		}

		inbound := singboxConfig.InboundsByTag["vless-non-string-sid"]
		if inbound["tls"] != "reality" {
			t.Errorf("Expected reality tls, got %v", inbound["tls"])
		}
		if inbound["sid"] != "" {
			t.Errorf("Expected empty sid for non-string short_id, got %v", inbound["sid"])
		}
	})
}

func TestAppendUser(t *testing.T) {
	config := `{
		"inbounds": [
			{
				"type": "vmess",
				"tag": "vmess-in",
				"listen_port": 1080,
				"users": [
					{
						"name": "existing.user",
						"uuid": "existing-uuid"
					}
				]
			},
			{
				"type": "shadowsocks",
				"tag": "ss-in",
				"listen_port": 1081,
				"users": []
			},
			{
				"type": "trojan",
				"tag": "trojan-in",
				"listen_port": 443
			}
		],
		"experimental": {
			"v2ray_api": {
				"listen": "127.0.0.1:8080",
				"stats": {
					"enabled": true,
					"users": ["existing.user"]
				}
			}
		}
	}`

	singboxConfig, err := NewSingBoxConfig(config, "127.0.0.1", 8080)
	if err != nil {
		t.Fatalf("Failed to create SingBoxConfig: %v", err)
	}

	user := models.User{
		ID:       12345,
		Username: "testuser",
		Key:      "test-key-123",
	}

	t.Run("append to existing users", func(t *testing.T) {
		inbound := models.Inbound{
			Tag:      "vmess-in",
			Protocol: "vmess",
		}

		err := singboxConfig.AppendUser(user, inbound)
		if err != nil {
			t.Fatalf("AppendUser failed: %v", err)
		}

		inbounds := singboxConfig.Data["inbounds"].([]any)
		vmessInbound := inbounds[0].(map[string]any)
		users := vmessInbound["users"].([]any)

		if len(users) != 2 {
			t.Errorf("Expected 2 users, got %d", len(users))
		}

		newUser := users[1].(map[string]any)
		if newUser["name"] != "12345.testuser" {
			t.Errorf("Expected name '12345.testuser', got %v", newUser["name"])
		}
	})

	t.Run("append to missing users field", func(t *testing.T) {
		inbound := models.Inbound{
			Tag:      "trojan-in",
			Protocol: "trojan",
		}

		user3 := models.User{
			ID:       99999,
			Username: "trojanuser",
			Key:      "trojan-password",
		}

		err := singboxConfig.AppendUser(user3, inbound)
		if err != nil {
			t.Fatalf("AppendUser failed: %v", err)
		}

		inbounds := singboxConfig.Data["inbounds"].([]any)
		trojanInbound := inbounds[2].(map[string]any)
		users := trojanInbound["users"].([]any)

		if len(users) != 1 {
			t.Errorf("Expected 1 user, got %d", len(users))
		}
	})

	t.Run("error cases", func(t *testing.T) {
		invalidInbound := models.Inbound{
			Tag:      "vmess-in",
			Protocol: "invalid-protocol",
		}

		err := singboxConfig.AppendUser(user, invalidInbound)
		if err == nil {
			t.Error("Expected error for invalid protocol, got nil")
		}

		noInboundsConfig := &SingBoxConfig{
			Data:          map[string]any{},
			ApiHost:       "127.0.0.1",
			ApiPort:       8080,
			Inbounds:      []map[string]any{},
			InboundsByTag: map[string]map[string]any{},
		}

		err = noInboundsConfig.AppendUser(user, models.Inbound{Tag: "test", Protocol: "vmess"})
		if err == nil || err.Error() != "inbounds not found in config" {
			t.Errorf("Expected 'inbounds not found in config' error, got: %v", err)
		}
	})
}

func TestPopUser(t *testing.T) {
	config := `{
		"inbounds": [
			{
				"type": "vmess",
				"tag": "vmess-in",
				"listen_port": 1080,
				"users": [
					{
						"name": "12345.testuser",
						"uuid": "test-uuid-1"
					},
					{
						"name": "67890.anotheruser",
						"uuid": "test-uuid-2"
					},
					{
						"username": "99999.usernameuser",
						"uuid": "test-uuid-3"
					}
				]
			},
			{
				"type": "shadowsocks",
				"tag": "ss-in",
				"listen_port": 1081,
				"users": [
					{
						"name": "11111.ssuser",
						"password": "ss-password"
					}
				]
			},
			{
				"type": "trojan",
				"tag": "trojan-in",
				"listen_port": 443,
				"users": []
			}
		]
	}`

	singboxConfig, err := NewSingBoxConfig(config, "127.0.0.1", 8080)
	if err != nil {
		t.Fatalf("Failed to create SingBoxConfig: %v", err)
	}

	t.Run("pop user by name", func(t *testing.T) {
		user := models.User{
			ID:       12345,
			Username: "testuser",
			Key:      "test-key",
		}

		inbound := models.Inbound{
			Tag:      "vmess-in",
			Protocol: "vmess",
		}

		err := singboxConfig.PopUser(user, inbound)
		if err != nil {
			t.Fatalf("PopUser failed: %v", err)
		}

		inbounds := singboxConfig.Data["inbounds"].([]any)
		vmessInbound := inbounds[0].(map[string]any)
		users := vmessInbound["users"].([]any)

		if len(users) != 2 {
			t.Errorf("Expected 2 users after pop, got %d", len(users))
		}

		for _, userItem := range users {
			userMap := userItem.(map[string]any)
			if name, ok := userMap["name"].(string); ok && name == "12345.testuser" {
				t.Error("User should have been removed but still found")
			}
		}
	})

	t.Run("pop last user", func(t *testing.T) {
		user := models.User{
			ID:       11111,
			Username: "ssuser",
			Key:      "test-key",
		}

		inbound := models.Inbound{
			Tag:      "ss-in",
			Protocol: "shadowsocks",
		}

		err := singboxConfig.PopUser(user, inbound)
		if err != nil {
			t.Fatalf("PopUser failed: %v", err)
		}

		inbounds := singboxConfig.Data["inbounds"].([]any)
		ssInbound := inbounds[1].(map[string]any)
		users := ssInbound["users"].([]any)

		if len(users) != 0 {
			t.Errorf("Expected 0 users after popping last user, got %d", len(users))
		}
	})

	t.Run("error cases", func(t *testing.T) {
		noInboundsConfig := &SingBoxConfig{
			Data: map[string]any{},
		}

		user := models.User{ID: 123, Username: "test", Key: "key"}
		inbound := models.Inbound{Tag: "test", Protocol: "vmess"}

		err := noInboundsConfig.PopUser(user, inbound)
		if err == nil || err.Error() != "inbounds not found in config" {
			t.Errorf("Expected 'inbounds not found in config' error, got: %v", err)
		}
	})

	t.Run("username field", func(t *testing.T) {
		config := &SingBoxConfig{
			Data: map[string]any{
				"inbounds": []any{
					map[string]any{
						"type":        "vmess",
						"tag":         "vmess-username-test",
						"listen_port": 1080,
						"users": []any{
							map[string]any{
								"username": "12345.testuser",
								"uuid":     "test-uuid",
							},
							map[string]any{
								"username": "67890.otheruser",
								"uuid":     "other-uuid",
							},
						},
					},
				},
			},
			ApiHost:       "127.0.0.1",
			ApiPort:       8080,
			Inbounds:      []map[string]any{},
			InboundsByTag: map[string]map[string]any{},
		}

		user := models.User{
			ID:       12345,
			Username: "testuser",
			Key:      "test-key",
		}

		inbound := models.Inbound{
			Tag:      "vmess-username-test",
			Protocol: "vmess",
		}

		err := config.PopUser(user, inbound)
		if err != nil {
			t.Fatalf("PopUser failed: %v", err)
		}

		inbounds := config.Data["inbounds"].([]any)
		vmessInbound := inbounds[0].(map[string]any)
		users := vmessInbound["users"].([]any)

		if len(users) != 1 {
			t.Errorf("Expected 1 user remaining after pop, got %d", len(users))
		}

		remainingUser := users[0].(map[string]any)
		if remainingUser["username"] != "67890.otheruser" {
			t.Errorf("Expected remaining user '67890.otheruser', got %v", remainingUser["username"])
		}
	})
}

func TestRegisterInbounds(t *testing.T) {
	config := `{
		"inbounds": [
			{
				"type": "shadowsocks",
				"tag": "ss-in",
				"listen_port": 1080,
				"users": []
			},
			{
				"type": "vmess",
				"tag": "vmess-in",
				"listen_port": 1081,
				"users": []
			}
		]
	}`

	singboxConfig, err := NewSingBoxConfig(config, "127.0.0.1", 8080)
	if err != nil {
		t.Fatalf("Failed to create SingBoxConfig: %v", err)
	}

	t.Run("successful registration", func(t *testing.T) {
		mockStorage := &MockStorage{
			inbounds: make(map[string]*models.Inbound),
		}

		err := singboxConfig.RegisterInbounds(mockStorage)
		if err != nil {
			t.Fatalf("RegisterInbounds failed: %v", err)
		}

		if len(mockStorage.inbounds) != 2 {
			t.Errorf("Expected 2 inbounds registered, got %d", len(mockStorage.inbounds))
		}

		if _, exists := mockStorage.inbounds["ss-in"]; !exists {
			t.Error("ss-in inbound not registered")
		}

		if _, exists := mockStorage.inbounds["vmess-in"]; !exists {
			t.Error("vmess-in inbound not registered")
		}
	})

	t.Run("storage error", func(t *testing.T) {
		mockStorage := &MockStorage{
			inbounds:    make(map[string]*models.Inbound),
			shouldError: true,
		}

		err := singboxConfig.RegisterInbounds(mockStorage)
		if err == nil {
			t.Error("Expected error from storage, got nil")
		}

		if !strings.Contains(err.Error(), "failed to register inbound") {
			t.Errorf("Expected error message about registration failure, got: %v", err)
		}
	})
}

func TestListInbounds(t *testing.T) {
	t.Run("with inbounds", func(t *testing.T) {
		config := `{
			"inbounds": [
				{
					"type": "shadowsocks",
					"tag": "ss-in",
					"listen_port": 1080,
					"users": []
				},
				{
					"type": "vmess",
					"tag": "vmess-in",
					"listen_port": 1081,
					"users": []
				}
			]
		}`

		singboxConfig, err := NewSingBoxConfig(config, "127.0.0.1", 8080)
		if err != nil {
			t.Fatalf("Failed to create SingBoxConfig: %v", err)
		}

		inbounds := singboxConfig.ListInbounds()
		if len(inbounds) != 2 {
			t.Errorf("Expected 2 inbounds from ListInbounds(), got %d", len(inbounds))
		}

		tagFound := make(map[string]bool)
		for _, inbound := range inbounds {
			tagFound[inbound.Tag] = true
			if inbound.Tag == "ss-in" && inbound.Protocol != "shadowsocks" {
				t.Errorf("Expected shadowsocks protocol for ss-in, got %s", inbound.Protocol)
			}
			if inbound.Tag == "vmess-in" && inbound.Protocol != "vmess" {
				t.Errorf("Expected vmess protocol for vmess-in, got %s", inbound.Protocol)
			}
		}

		if !tagFound["ss-in"] {
			t.Error("ss-in tag not found in ListInbounds() result")
		}
		if !tagFound["vmess-in"] {
			t.Error("vmess-in tag not found in ListInbounds() result")
		}
	})

	t.Run("empty config", func(t *testing.T) {
		config := &SingBoxConfig{
			InboundsByTag: make(map[string]map[string]interface{}),
		}

		inbounds := config.ListInbounds()
		if len(inbounds) != 0 {
			t.Errorf("expected 0 inbounds, got %d", len(inbounds))
		}
	})

	t.Run("edge cases", func(t *testing.T) {
		config := &SingBoxConfig{
			InboundsByTag: map[string]map[string]interface{}{
				"test-key": {
					"tag":      123,
					"protocol": "vmess",
				},
				"test-key2": {
					"tag":      "test-tag",
					"protocol": 456,
				},
			},
		}

		inbounds := config.ListInbounds()
		if len(inbounds) != 2 {
			t.Errorf("expected 2 inbounds, got %d", len(inbounds))
		}

		for _, inbound := range inbounds {
			if inbound.Tag == "" && inbound.Protocol != "vmess" {
				t.Error("Expected vmess protocol for inbound with non-string tag")
			}
			if inbound.Tag == "test-tag" && inbound.Protocol != "" {
				t.Error("Expected empty protocol for inbound with non-string protocol")
			}
		}
	})
}

func TestToJSON(t *testing.T) {
	t.Run("basic serialization", func(t *testing.T) {
		config := &SingBoxConfig{
			Data: map[string]any{
				"log": map[string]any{
					"level": "info",
				},
				"inbounds": []any{
					map[string]any{
						"tag":  "test",
						"type": "shadowsocks",
					},
				},
			},
			ApiHost: "127.0.0.1",
			ApiPort: 8080,
			Inbounds: []map[string]any{
				{
					"tag":      "test",
					"protocol": "shadowsocks",
				},
			},
			InboundsByTag: map[string]map[string]any{
				"test": {
					"tag":      "test",
					"protocol": "shadowsocks",
				},
			},
		}

		result, err := config.ToJSON()
		if err != nil {
			t.Fatalf("ToJSON failed: %v", err)
		}

		if !strings.Contains(result, `"apiHost":"127.0.0.1"`) {
			t.Errorf("Expected result to contain apiHost")
		}
		if !strings.Contains(result, `"apiPort":8080`) {
			t.Errorf("Expected result to contain apiPort")
		}
		if !strings.Contains(result, `"shadowsocks"`) {
			t.Errorf("Expected result to contain protocol")
		}
	})

	t.Run("empty config", func(t *testing.T) {
		config := &SingBoxConfig{
			Data:          map[string]any{},
			ApiHost:       "",
			ApiPort:       0,
			Inbounds:      []map[string]any{},
			InboundsByTag: map[string]map[string]any{},
		}

		result, err := config.ToJSON()
		if err != nil {
			t.Fatalf("ToJSON failed: %v", err)
		}

		if !strings.Contains(result, `"data":{}`) {
			t.Errorf("Expected result to contain empty data object")
		}
	})

	t.Run("error cases", func(t *testing.T) {
		config := &SingBoxConfig{
			Data: map[string]any{
				"invalid": make(chan int),
			},
			ApiHost:       "127.0.0.1",
			ApiPort:       8080,
			Inbounds:      []map[string]any{},
			InboundsByTag: map[string]map[string]any{},
		}

		_, err := config.ToJSON()
		if err == nil {
			t.Error("Expected error for unmarshalable data, got nil")
		}
		if !strings.Contains(err.Error(), "failed to encode to JSON") {
			t.Errorf("Expected JSON encoding error, got: %v", err)
		}
	})
}

type MockStorage struct {
	inbounds    map[string]*models.Inbound
	shouldError bool
}

func (m *MockStorage) RegisterInbound(inbound models.Inbound) error {
	if m.shouldError {
		return fmt.Errorf("mock storage error")
	}
	m.inbounds[inbound.Tag] = &inbound
	return nil
}

func (m *MockStorage) ListUsers(userID *int64) ([]models.User, error) { return nil, nil }
func (m *MockStorage) ListInbounds(tags []string, includeUsers bool) ([]models.Inbound, error) {
	return nil, nil
}
func (m *MockStorage) ListInboundUsers(tag string) ([]models.User, error) { return nil, nil }
func (m *MockStorage) RemoveUser(user models.User) error                  { return nil }
func (m *MockStorage) UpdateUserInbounds(user models.User, inbounds []models.Inbound) error {
	return nil
}
func (m *MockStorage) RemoveInbound(inbound models.Inbound) error { return nil }
func (m *MockStorage) FlushUsers() error                          { return nil }

// =============================================================================
// Additional Tests for 100% Coverage
// =============================================================================

func TestNewSingBoxConfig_FileParsingError(t *testing.T) {
	tmpFile := "/tmp/invalid_json_file.json"
	invalidJSON := `{"invalid": json content}`
	err := os.WriteFile(tmpFile, []byte(invalidJSON), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile)

	_, err = NewSingBoxConfig(tmpFile, "127.0.0.1", 8080)
	if err == nil {
		t.Error("Expected error for invalid JSON file, got nil")
	}
	if !strings.Contains(err.Error(), "failed to parse file content as JSON") {
		t.Errorf("Expected file parsing error, got: %v", err)
	}
}

func TestResolveInbounds_InvalidInboundTypes(t *testing.T) {
	config := &SingBoxConfig{
		Data: map[string]any{
			"inbounds": []any{
				"invalid_string_inbound",
				map[string]any{
					"type":        "shadowtls",
					"tag":         "shadowtls-with-version",
					"listen_port": 443,
					"version":     3,
				},
			},
		},
		Inbounds:      make([]map[string]any, 0),
		InboundsByTag: make(map[string]map[string]any),
	}

	config.resolveInbounds()

	if len(config.InboundsByTag) != 1 {
		t.Errorf("Expected 1 valid inbound, got %d", len(config.InboundsByTag))
	}

	shadowtlsInbound := config.InboundsByTag["shadowtls-with-version"]
	if shadowtlsInbound["shadowtls_version"] != 3 {
		t.Errorf("Expected shadowtls_version 3, got %v", shadowtlsInbound["shadowtls_version"])
	}
}

func TestAppendUser_InvalidDataStructures(t *testing.T) {
	config := &SingBoxConfig{
		Data: map[string]any{
			"inbounds": []any{
				"invalid_string_item",
				map[string]any{
					"type":        "vmess",
					"tag":         "vmess-test",
					"listen_port": 1080,
					"users":       []any{},
				},
			},
			"experimental": map[string]any{
				"v2ray_api": map[string]any{
					"stats": map[string]any{
						"enabled": true,
						"users":   "invalid_non_array_users",
					},
				},
			},
		},
		ApiHost:       "127.0.0.1",
		ApiPort:       8080,
		Inbounds:      []map[string]any{},
		InboundsByTag: map[string]map[string]any{},
	}

	user := models.User{
		ID:       12345,
		Username: "testuser",
		Key:      "test-key",
	}

	inbound := models.Inbound{
		Tag:      "vmess-test",
		Protocol: "vmess",
	}

	err := config.AppendUser(user, inbound)
	if err != nil {
		t.Fatalf("AppendUser failed: %v", err)
	}

	experimental := config.Data["experimental"].(map[string]any)
	v2rayAPI := experimental["v2ray_api"].(map[string]any)
	stats := v2rayAPI["stats"].(map[string]any)

	users, ok := stats["users"].([]string)
	if !ok {
		t.Fatal("Expected users to be converted to []string")
	}

	if len(users) != 1 || users[0] != "12345.testuser" {
		t.Errorf("Expected users ['12345.testuser'], got %v", users)
	}
}

func TestPopUser_InvalidDataStructures(t *testing.T) {
	config := &SingBoxConfig{
		Data: map[string]any{
			"inbounds": []any{
				"invalid_string_item",
				map[string]any{
					"type":        "vmess",
					"tag":         "vmess-without-users",
					"listen_port": 1080,
				},
				map[string]any{
					"type":        "vmess",
					"tag":         "vmess-with-invalid-users",
					"listen_port": 1081,
					"users": []any{
						"invalid_string_user",
						map[string]any{
							"name": "different.user",
							"uuid": "test-uuid",
						},
					},
				},
			},
		},
		ApiHost:       "127.0.0.1",
		ApiPort:       8080,
		Inbounds:      []map[string]any{},
		InboundsByTag: map[string]map[string]any{},
	}

	user := models.User{
		ID:       12345,
		Username: "testuser",
		Key:      "test-key",
	}

	t.Run("inbound without users field", func(t *testing.T) {
		inbound := models.Inbound{
			Tag:      "vmess-without-users",
			Protocol: "vmess",
		}

		err := config.PopUser(user, inbound)
		if err != nil {
			t.Fatalf("PopUser failed: %v", err)
		}
	})

	t.Run("inbound with invalid user structures", func(t *testing.T) {
		inbound := models.Inbound{
			Tag:      "vmess-with-invalid-users",
			Protocol: "vmess",
		}

		err := config.PopUser(user, inbound)
		if err != nil {
			t.Fatalf("PopUser failed: %v", err)
		}

		inbounds := config.Data["inbounds"].([]any)
		vmessInbound := inbounds[2].(map[string]any)
		users := vmessInbound["users"].([]any)

		if len(users) != 1 {
			t.Errorf("Expected 1 user remaining, got %d", len(users))
		}

		remainingUser := users[0].(map[string]any)
		if remainingUser["name"] != "different.user" {
			t.Errorf("Expected remaining user 'different.user', got %v", remainingUser["name"])
		}
	})
}
