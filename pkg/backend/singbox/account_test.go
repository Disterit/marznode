package singbox

import (
	"strings"
	"testing"

	"github.com/highlight-apps/node-backend/config"
)

func TestXTLSFlowConstants(t *testing.T) {
	if XTLSFlowNone != "" {
		t.Errorf("Expected XTLSFlowNone to be empty string, got %s", XTLSFlowNone)
	}

	if XTLSFlowVision != "xtls-rprx-vision" {
		t.Errorf("Expected XTLSFlowVision to be 'xtls-rprx-vision', got %s", XTLSFlowVision)
	}
}

func TestSingboxAccount(t *testing.T) {
	account := SingboxAccount{
		Identifier: "test-user",
		Seed:       "test-seed",
	}

	if account.GetIdentifier() != "test-user" {
		t.Errorf("Expected identifier 'test-user', got %s", account.GetIdentifier())
	}

	if account.GetSeed() != "test-seed" {
		t.Errorf("Expected seed 'test-seed', got %s", account.GetSeed())
	}

	str := account.String()
	if !strings.Contains(str, "test-user") {
		t.Errorf("String representation should contain identifier, got %s", str)
	}
	if !strings.Contains(str, "SingboxAccount") {
		t.Errorf("String representation should contain type name, got %s", str)
	}
}

func TestNamedAccount(t *testing.T) {
	t.Run("Constructor", func(t *testing.T) {
		account := NewNamedAccount("test-user", "test-seed")

		if account.Name != "test-user" {
			t.Errorf("Expected name 'test-user', got %s", account.Name)
		}

		if account.GetIdentifier() != "test-user" {
			t.Errorf("Expected identifier 'test-user', got %s", account.GetIdentifier())
		}

		if account.GetSeed() != "test-seed" {
			t.Errorf("Expected seed 'test-seed', got %s", account.GetSeed())
		}
	})

	t.Run("ToDict", func(t *testing.T) {
		account := NewNamedAccount("test-user", "test-seed")
		dict := account.ToDict()

		if dict["name"] != "test-user" {
			t.Errorf("Expected ToDict name 'test-user', got %v", dict["name"])
		}
	})

	t.Run("String", func(t *testing.T) {
		account := NewNamedAccount("test-user", "test-seed")
		str := account.String()

		if !strings.Contains(str, "test-user") {
			t.Errorf("String representation should contain identifier, got %s", str)
		}
	})
}

func TestUserNamedAccount(t *testing.T) {
	t.Run("Constructor", func(t *testing.T) {
		account := NewUserNamedAccount("test-user", "test-seed")

		if account.Username != "test-user" {
			t.Errorf("Expected username 'test-user', got %s", account.Username)
		}

		if account.GetIdentifier() != "test-user" {
			t.Errorf("Expected identifier 'test-user', got %s", account.GetIdentifier())
		}

		if account.GetSeed() != "test-seed" {
			t.Errorf("Expected seed 'test-seed', got %s", account.GetSeed())
		}
	})

	t.Run("ToDict", func(t *testing.T) {
		account := NewUserNamedAccount("test-user", "test-seed")
		dict := account.ToDict()

		if dict["username"] != "test-user" {
			t.Errorf("Expected ToDict username 'test-user', got %v", dict["username"])
		}
	})
}

func TestVMessAccount(t *testing.T) {
	t.Run("Auto generation", func(t *testing.T) {
		account, err := NewVMessAccount("test-user", "test-seed", "")
		if err != nil {
			t.Fatalf("Failed to create VMessAccount: %v", err)
		}

		if account.Name != "test-user" {
			t.Errorf("Expected name 'test-user', got %s", account.Name)
		}

		if account.UUID == "" {
			t.Error("UUID should be auto-generated when empty")
		}

		dict := account.ToDict()
		expectedKeys := []string{"name", "uuid"}
		for _, key := range expectedKeys {
			if _, exists := dict[key]; !exists {
				t.Errorf("ToDict should include key '%s'", key)
			}
		}

		if dict["name"] != "test-user" {
			t.Errorf("Expected ToDict name 'test-user', got %v", dict["name"])
		}
		if dict["uuid"] == "" {
			t.Error("ToDict should include generated UUID")
		}
	})

	t.Run("Preset UUID", func(t *testing.T) {
		customUUID := "custom-uuid-123"
		account, err := NewVMessAccount("test-user", "test-seed", customUUID)
		if err != nil {
			t.Fatalf("Failed to create VMessAccount: %v", err)
		}

		if account.UUID != customUUID {
			t.Errorf("Expected UUID '%s', got %s", customUUID, account.UUID)
		}
	})

	t.Run("Empty seed", func(t *testing.T) {
		_, err := NewVMessAccount("test-user", "", "")
		if err == nil {
			t.Error("Expected error for empty seed")
		}
		if !strings.Contains(err.Error(), "seed cannot be empty") {
			t.Errorf("Expected 'seed cannot be empty' error, got %v", err)
		}
	})

	t.Run("Seed consistency", func(t *testing.T) {
		seed := "consistent-seed"
		account1, err1 := NewVMessAccount("user", seed, "")
		account2, err2 := NewVMessAccount("user", seed, "")

		if err1 != nil || err2 != nil {
			t.Fatalf("Failed to create accounts: %v, %v", err1, err2)
		}

		if account1.UUID != account2.UUID {
			t.Error("Same seed should generate same UUID")
		}
	})
}

func TestVLESSAccount(t *testing.T) {
	t.Run("Auto generation", func(t *testing.T) {
		account, err := NewVLESSAccount("test-user", "test-seed", "", XTLSFlowNone)
		if err != nil {
			t.Fatalf("Failed to create VLESSAccount: %v", err)
		}

		if account.UUID == "" {
			t.Error("UUID should be auto-generated when empty")
		}

		if account.Flow != XTLSFlowNone {
			t.Errorf("Expected flow %s, got %s", XTLSFlowNone, account.Flow)
		}

		dict := account.ToDict()
		expectedKeys := []string{"name", "uuid", "flow"}
		for _, key := range expectedKeys {
			if _, exists := dict[key]; !exists {
				t.Errorf("ToDict should include key '%s'", key)
			}
		}

		if dict["flow"] != XTLSFlowNone {
			t.Errorf("Expected ToDict flow %s, got %v", XTLSFlowNone, dict["flow"])
		}
	})

	t.Run("Preset UUID and flow", func(t *testing.T) {
		customUUID := "custom-uuid-456"
		account, err := NewVLESSAccount("test-user", "test-seed", customUUID, XTLSFlowVision)
		if err != nil {
			t.Fatalf("Failed to create VLESSAccount: %v", err)
		}

		if account.UUID != customUUID {
			t.Errorf("Expected UUID '%s', got %s", customUUID, account.UUID)
		}

		if account.Flow != XTLSFlowVision {
			t.Errorf("Expected flow %s, got %s", XTLSFlowVision, account.Flow)
		}
	})

	t.Run("Empty seed", func(t *testing.T) {
		_, err := NewVLESSAccount("test-user", "", "custom-uuid", XTLSFlowNone)
		if err == nil {
			t.Error("Expected error for empty seed")
		}
	})

	t.Run("Flow field in ToDict", func(t *testing.T) {
		account, err := NewVLESSAccount("test-user", "test-seed", "", XTLSFlowNone)
		if err != nil {
			t.Fatalf("Failed to create VLESSAccount: %v", err)
		}

		dict := account.ToDict()
		if dict["flow"] != XTLSFlowNone {
			t.Errorf("Expected ToDict flow %s, got %v", XTLSFlowNone, dict["flow"])
		}
	})
}

func TestTrojanAccount(t *testing.T) {
	t.Run("Auto generation", func(t *testing.T) {
		account, err := NewTrojanAccount("test-user", "test-seed", "")
		if err != nil {
			t.Fatalf("Failed to create TrojanAccount: %v", err)
		}

		if account.Password == "" {
			t.Error("Password should be auto-generated when empty")
		}

		dict := account.ToDict()
		expectedKeys := []string{"name", "password"}
		for _, key := range expectedKeys {
			if _, exists := dict[key]; !exists {
				t.Errorf("ToDict should include key '%s'", key)
			}
		}

		if dict["password"] == "" {
			t.Error("ToDict should include generated password")
		}
	})

	t.Run("Preset password", func(t *testing.T) {
		customPassword := "custom-password-123"
		account, err := NewTrojanAccount("test-user", "test-seed", customPassword)
		if err != nil {
			t.Fatalf("Failed to create TrojanAccount: %v", err)
		}

		if account.Password != customPassword {
			t.Errorf("Expected password '%s', got %s", customPassword, account.Password)
		}
	})

	t.Run("Empty seed", func(t *testing.T) {
		_, err := NewTrojanAccount("test-user", "", "custom-password")
		if err == nil {
			t.Error("Expected error for empty seed")
		}
	})
}

func TestShadowsocksAccount(t *testing.T) {
	t.Run("Auto generation", func(t *testing.T) {
		account, err := NewShadowsocksAccount("test-user", "test-seed", "")
		if err != nil {
			t.Fatalf("Failed to create ShadowsocksAccount: %v", err)
		}

		dict := account.ToDict()
		if dict["name"] != "test-user" {
			t.Errorf("Expected name 'test-user', got %v", dict["name"])
		}

		if dict["password"] == "" {
			t.Error("Password should be auto-generated")
		}
	})

	t.Run("Preset password", func(t *testing.T) {
		customPassword := "custom-password-123"
		account, err := NewShadowsocksAccount("test-user", "test-seed", customPassword)
		if err != nil {
			t.Fatalf("Failed to create ShadowsocksAccount: %v", err)
		}

		dict := account.ToDict()
		if dict["password"] != customPassword {
			t.Errorf("Expected preset password '%s', got %v", customPassword, dict["password"])
		}
	})

	t.Run("Empty seed", func(t *testing.T) {
		_, err := NewShadowsocksAccount("test-user", "", "password")
		if err == nil {
			t.Error("Expected error for empty seed")
		}
	})
}

func TestTUICAccount(t *testing.T) {
	t.Run("Auto generation", func(t *testing.T) {
		account, err := NewTUICAccount("test-user", "test-seed", "", "")
		if err != nil {
			t.Fatalf("Failed to create TUICAccount: %v", err)
		}

		if account.UUID == "" {
			t.Error("UUID should be auto-generated when empty")
		}

		if account.Password == "" {
			t.Error("Password should be auto-generated when empty")
		}

		dict := account.ToDict()
		expectedKeys := []string{"name", "uuid", "password"}
		for _, key := range expectedKeys {
			if _, exists := dict[key]; !exists {
				t.Errorf("ToDict should include key '%s'", key)
			}
		}

		if dict["uuid"] == "" || dict["password"] == "" {
			t.Error("ToDict should include generated UUID and password")
		}
	})

	t.Run("Preset credentials", func(t *testing.T) {
		customUUID := "custom-uuid-789"
		customPassword := "custom-password-789"
		account, err := NewTUICAccount("test-user", "test-seed", customUUID, customPassword)
		if err != nil {
			t.Fatalf("Failed to create TUICAccount: %v", err)
		}

		if account.UUID != customUUID {
			t.Errorf("Expected UUID '%s', got %s", customUUID, account.UUID)
		}

		if account.Password != customPassword {
			t.Errorf("Expected password '%s', got %s", customPassword, account.Password)
		}
	})

	t.Run("Partial preset", func(t *testing.T) {
		customUUID := "custom-uuid-only"
		account, err := NewTUICAccount("test-user", "test-seed", customUUID, "")
		if err != nil {
			t.Fatalf("Failed to create TUICAccount: %v", err)
		}

		if account.UUID != customUUID {
			t.Errorf("Expected UUID '%s', got %s", customUUID, account.UUID)
		}

		if account.Password == "" {
			t.Error("Password should be auto-generated when empty")
		}
	})

	t.Run("Empty seed", func(t *testing.T) {
		_, err := NewTUICAccount("test-user", "", "custom-uuid", "custom-password")
		if err == nil {
			t.Error("Expected error for empty seed")
		}
	})
}

func TestHysteria2Account(t *testing.T) {
	t.Run("Auto generation", func(t *testing.T) {
		account, err := NewHysteria2Account("test-user", "test-seed", "")
		if err != nil {
			t.Fatalf("Failed to create Hysteria2Account: %v", err)
		}

		dict := account.ToDict()
		if dict["name"] != "test-user" {
			t.Errorf("Expected name 'test-user', got %v", dict["name"])
		}

		if dict["password"] == "" {
			t.Error("Password should be auto-generated")
		}
	})

	t.Run("Preset password", func(t *testing.T) {
		customPassword := "custom-password-123"
		account, err := NewHysteria2Account("test-user", "test-seed", customPassword)
		if err != nil {
			t.Fatalf("Failed to create Hysteria2Account: %v", err)
		}

		dict := account.ToDict()
		if dict["password"] != customPassword {
			t.Errorf("Expected preset password '%s', got %v", customPassword, dict["password"])
		}
	})

	t.Run("Empty seed", func(t *testing.T) {
		_, err := NewHysteria2Account("test-user", "", "password")
		if err == nil {
			t.Error("Expected error for empty seed")
		}
	})
}

func TestNaiveAccount(t *testing.T) {
	t.Run("Auto generation", func(t *testing.T) {
		account, err := NewNaiveAccount("test-user", "test-seed", "")
		if err != nil {
			t.Fatalf("Failed to create NaiveAccount: %v", err)
		}

		dict := account.ToDict()
		if dict["username"] != "test-user" {
			t.Errorf("Expected username 'test-user', got %v", dict["username"])
		}

		if dict["password"] == "" {
			t.Error("Password should be auto-generated")
		}
	})

	t.Run("Preset password", func(t *testing.T) {
		customPassword := "custom-password-456"
		account, err := NewNaiveAccount("test-user", "test-seed", customPassword)
		if err != nil {
			t.Fatalf("Failed to create NaiveAccount: %v", err)
		}

		dict := account.ToDict()
		if dict["password"] != customPassword {
			t.Errorf("Expected preset password '%s', got %v", customPassword, dict["password"])
		}
	})

	t.Run("Empty seed", func(t *testing.T) {
		_, err := NewNaiveAccount("test-user", "", "password")
		if err == nil {
			t.Error("Expected error for empty seed")
		}
	})
}

func TestShadowTLSAccount(t *testing.T) {
	t.Run("Auto generation", func(t *testing.T) {
		account, err := NewShadowTLSAccount("test-user", "test-seed", "")
		if err != nil {
			t.Fatalf("Failed to create ShadowTLSAccount: %v", err)
		}

		dict := account.ToDict()
		if dict["name"] != "test-user" {
			t.Errorf("Expected name 'test-user', got %v", dict["name"])
		}

		if dict["password"] == "" {
			t.Error("Password should be auto-generated")
		}
	})

	t.Run("Preset password", func(t *testing.T) {
		customPassword := "custom-password-123"
		account, err := NewShadowTLSAccount("test-user", "test-seed", customPassword)
		if err != nil {
			t.Fatalf("Failed to create ShadowTLSAccount: %v", err)
		}

		dict := account.ToDict()
		if dict["password"] != customPassword {
			t.Errorf("Expected preset password '%s', got %v", customPassword, dict["password"])
		}
	})

	t.Run("Empty seed", func(t *testing.T) {
		_, err := NewShadowTLSAccount("test-user", "", "password")
		if err == nil {
			t.Error("Expected error for empty seed")
		}
	})
}

func TestSocksAccount(t *testing.T) {
	t.Run("Auto generation", func(t *testing.T) {
		account, err := NewSocksAccount("test-user", "test-seed", "")
		if err != nil {
			t.Fatalf("Failed to create SocksAccount: %v", err)
		}

		dict := account.ToDict()
		if dict["username"] != "test-user" {
			t.Errorf("Expected username 'test-user', got %v", dict["username"])
		}

		if dict["password"] == "" {
			t.Error("Password should be auto-generated")
		}
	})

	t.Run("Preset password", func(t *testing.T) {
		customPassword := "custom-password-456"
		account, err := NewSocksAccount("test-user", "test-seed", customPassword)
		if err != nil {
			t.Fatalf("Failed to create SocksAccount: %v", err)
		}

		dict := account.ToDict()
		if dict["password"] != customPassword {
			t.Errorf("Expected preset password '%s', got %v", customPassword, dict["password"])
		}
	})

	t.Run("Empty seed", func(t *testing.T) {
		_, err := NewSocksAccount("test-user", "", "password")
		if err == nil {
			t.Error("Expected error for empty seed")
		}
	})
}

func TestHTTPAccount(t *testing.T) {
	t.Run("Auto generation", func(t *testing.T) {
		account, err := NewHTTPAccount("test-user", "test-seed", "")
		if err != nil {
			t.Fatalf("Failed to create HTTPAccount: %v", err)
		}

		dict := account.ToDict()
		if dict["username"] != "test-user" {
			t.Errorf("Expected username 'test-user', got %v", dict["username"])
		}

		if dict["password"] == "" {
			t.Error("Password should be auto-generated")
		}
	})

	t.Run("Preset password", func(t *testing.T) {
		customPassword := "custom-password-456"
		account, err := NewHTTPAccount("test-user", "test-seed", customPassword)
		if err != nil {
			t.Fatalf("Failed to create HTTPAccount: %v", err)
		}

		dict := account.ToDict()
		if dict["password"] != customPassword {
			t.Errorf("Expected preset password '%s', got %v", customPassword, dict["password"])
		}
	})

	t.Run("Empty seed", func(t *testing.T) {
		_, err := NewHTTPAccount("test-user", "", "password")
		if err == nil {
			t.Error("Expected error for empty seed")
		}
	})
}

func TestMixedAccount(t *testing.T) {
	t.Run("Auto generation", func(t *testing.T) {
		account, err := NewMixedAccount("test-user", "test-seed", "")
		if err != nil {
			t.Fatalf("Failed to create MixedAccount: %v", err)
		}

		dict := account.ToDict()
		if dict["username"] != "test-user" {
			t.Errorf("Expected username 'test-user', got %v", dict["username"])
		}

		if dict["password"] == "" {
			t.Error("Password should be auto-generated")
		}
	})

	t.Run("Preset password", func(t *testing.T) {
		customPassword := "custom-password-456"
		account, err := NewMixedAccount("test-user", "test-seed", customPassword)
		if err != nil {
			t.Fatalf("Failed to create MixedAccount: %v", err)
		}

		dict := account.ToDict()
		if dict["password"] != customPassword {
			t.Errorf("Expected preset password '%s', got %v", customPassword, dict["password"])
		}
	})

	t.Run("Empty seed", func(t *testing.T) {
		_, err := NewMixedAccount("test-user", "", "password")
		if err == nil {
			t.Error("Expected error for empty seed")
		}
	})
}

func TestAccountOptions(t *testing.T) {
	t.Run("Empty AccountOptions", func(t *testing.T) {
		opts := &AccountOptions{}
		if opts.UUID != "" {
			t.Errorf("Expected empty UUID, got %s", opts.UUID)
		}
		if opts.Password != "" {
			t.Errorf("Expected empty Password, got %s", opts.Password)
		}
		if opts.Flow != "" {
			t.Errorf("Expected empty Flow, got %s", opts.Flow)
		}
	})

	t.Run("Filled AccountOptions", func(t *testing.T) {
		opts := &AccountOptions{
			UUID:     "test-uuid",
			Password: "test-password",
			Flow:     XTLSFlowVision,
		}
		if opts.UUID != "test-uuid" {
			t.Errorf("Expected UUID 'test-uuid', got %s", opts.UUID)
		}
		if opts.Password != "test-password" {
			t.Errorf("Expected Password 'test-password', got %s", opts.Password)
		}
		if opts.Flow != XTLSFlowVision {
			t.Errorf("Expected Flow %s, got %s", XTLSFlowVision, opts.Flow)
		}
	})
}

func TestAccountsMap(t *testing.T) {
	expectedProtocols := []string{
		"shadowsocks", "trojan", "vmess", "vless", "shadowtls",
		"tuic", "hysteria2", "naive", "socks", "mixed", "http",
	}

	t.Run("All protocols exist", func(t *testing.T) {
		for _, protocol := range expectedProtocols {
			if _, exists := AccountsMap[protocol]; !exists {
				t.Errorf("Protocol %s not found in AccountsMap", protocol)
			}
		}
	})

	t.Run("Factory functions work", func(t *testing.T) {
		for _, protocol := range expectedProtocols {
			t.Run(protocol, func(t *testing.T) {
				factory, exists := AccountsMap[protocol]
				if !exists {
					t.Errorf("Protocol %s not found in AccountsMap", protocol)
					return
				}

				account, err := factory("test-user", "test-seed", nil)
				if err != nil {
					t.Errorf("Failed to create account for protocol %s: %v", protocol, err)
					return
				}

				if account.GetIdentifier() != "test-user" {
					t.Errorf("Expected identifier 'test-user', got %s", account.GetIdentifier())
				}

				if account.GetSeed() != "test-seed" {
					t.Errorf("Expected seed 'test-seed', got %s", account.GetSeed())
				}

				dict := account.ToDict()
				if dict == nil {
					t.Error("ToDict should not return nil")
				}
			})
		}
	})

	t.Run("Factory functions with options", func(t *testing.T) {
		testCases := []struct {
			protocol string
			opts     *AccountOptions
		}{
			{"vmess", &AccountOptions{UUID: "test-uuid"}},
			{"vless", &AccountOptions{UUID: "test-uuid", Flow: XTLSFlowVision}},
			{"tuic", &AccountOptions{UUID: "test-uuid", Password: "test-pass"}},
			{"trojan", &AccountOptions{Password: "test-pass"}},
			{"shadowsocks", &AccountOptions{Password: "test-pass"}},
			{"hysteria2", &AccountOptions{Password: "test-pass"}},
			{"shadowtls", &AccountOptions{Password: "test-pass"}},
			{"naive", &AccountOptions{Password: "test-pass"}},
			{"socks", &AccountOptions{Password: "test-pass"}},
			{"http", &AccountOptions{Password: "test-pass"}},
			{"mixed", &AccountOptions{Password: "test-pass"}},
		}

		for _, tc := range testCases {
			t.Run(tc.protocol+"_with_options", func(t *testing.T) {
				factory := AccountsMap[tc.protocol]
				account, err := factory("test-user", "test-seed", tc.opts)
				if err != nil {
					t.Errorf("Failed to create account for protocol %s with options: %v", tc.protocol, err)
					return
				}

				dict := account.ToDict()
				if tc.opts.UUID != "" {
					if dict["uuid"] != tc.opts.UUID {
						t.Errorf("Expected UUID '%s', got %v", tc.opts.UUID, dict["uuid"])
					}
				}

				if tc.opts.Password != "" {
					if dict["password"] != tc.opts.Password {
						t.Errorf("Expected password '%s', got %v", tc.opts.Password, dict["password"])
					}
				}

				if tc.opts.Flow != "" {
					if dict["flow"] != tc.opts.Flow {
						t.Errorf("Expected flow '%s', got %v", tc.opts.Flow, dict["flow"])
					}
				}
			})
		}
	})
}

func TestCreateAccount(t *testing.T) {
	t.Run("Valid protocol", func(t *testing.T) {
		account, err := CreateAccount("vmess", "test-user", "test-seed", nil)
		if err != nil {
			t.Fatalf("Failed to create account: %v", err)
		}

		if account.GetIdentifier() != "test-user" {
			t.Errorf("Expected identifier 'test-user', got %s", account.GetIdentifier())
		}

		vmessAccount, ok := account.(*VMessAccount)
		if !ok {
			t.Error("Expected VMessAccount type")
		} else if vmessAccount.UUID == "" {
			t.Error("UUID should be generated")
		}
	})

	t.Run("Invalid protocol", func(t *testing.T) {
		_, err := CreateAccount("invalid-protocol", "test-user", "test-seed", nil)
		if err == nil {
			t.Error("Expected error for invalid protocol")
		}

		if !strings.Contains(err.Error(), "unsupported protocol") {
			t.Errorf("Expected 'unsupported protocol' error, got %v", err)
		}
	})

	t.Run("With options", func(t *testing.T) {
		opts := &AccountOptions{
			Flow: XTLSFlowVision,
			UUID: "custom-uuid",
		}
		account, err := CreateAccount("vless", "test-user", "test-seed", opts)
		if err != nil {
			t.Fatalf("Failed to create account with options: %v", err)
		}

		vlessAccount, ok := account.(*VLESSAccount)
		if !ok {
			t.Error("Expected VLESSAccount type")
		} else {
			if vlessAccount.UUID != "custom-uuid" {
				t.Errorf("Expected UUID 'custom-uuid', got %s", vlessAccount.UUID)
			}
			if vlessAccount.Flow != XTLSFlowVision {
				t.Errorf("Expected flow %s, got %s", XTLSFlowVision, vlessAccount.Flow)
			}
		}
	})

	t.Run("Nil options", func(t *testing.T) {
		account, err := CreateAccount("trojan", "test-user", "test-seed", nil)
		if err != nil {
			t.Fatalf("Failed to create account with nil options: %v", err)
		}

		if account.GetIdentifier() != "test-user" {
			t.Errorf("Expected identifier 'test-user', got %s", account.GetIdentifier())
		}
	})

	t.Run("Empty options", func(t *testing.T) {
		opts := &AccountOptions{}
		account, err := CreateAccount("vless", "test-user", "test-seed", opts)
		if err != nil {
			t.Fatalf("Failed to create account with empty options: %v", err)
		}

		vlessAccount, ok := account.(*VLESSAccount)
		if !ok {
			t.Error("Expected VLESSAccount type")
		} else {
			if vlessAccount.Flow != XTLSFlowNone {
				t.Errorf("Expected default flow %s, got %s", XTLSFlowNone, vlessAccount.Flow)
			}
		}
	})

	t.Run("All protocols", func(t *testing.T) {
		protocols := []string{
			"shadowsocks", "trojan", "vmess", "vless", "shadowtls",
			"tuic", "hysteria2", "naive", "socks", "mixed", "http",
		}

		for _, protocol := range protocols {
			t.Run(protocol, func(t *testing.T) {
				account, err := CreateAccount(protocol, "test-user", "test-seed", nil)
				if err != nil {
					t.Errorf("Failed to create %s account: %v", protocol, err)
					return
				}

				if account.GetIdentifier() != "test-user" {
					t.Errorf("Expected identifier 'test-user', got %s", account.GetIdentifier())
				}

				dict := account.ToDict()
				if dict == nil {
					t.Error("ToDict should not return nil")
				}
			})
		}
	})
}

func TestValidateAndGenerateFields(t *testing.T) {
	t.Run("Empty seed error", func(t *testing.T) {
		account := &VMessAccount{
			NamedAccount: *NewNamedAccount("test", ""),
		}

		err := validateAndGenerateFields(account, "")
		if err == nil {
			t.Error("Expected error for empty seed")
		}
		if !strings.Contains(err.Error(), "seed cannot be empty") {
			t.Errorf("Expected 'seed cannot be empty' error, got %v", err)
		}
	})

	t.Run("UUID generation", func(t *testing.T) {
		account := &VMessAccount{
			NamedAccount: *NewNamedAccount("test", "test-seed"),
		}

		err := validateAndGenerateFields(account, "test-seed")
		if err != nil {
			t.Fatalf("validateAndGenerateFields failed: %v", err)
		}

		if account.UUID == "" {
			t.Error("UUID should be generated")
		}
	})

	t.Run("UUID generation error", func(t *testing.T) {
		originalAlgorithm := config.GetAuthGenerationAlgorithm()
		config.SetAuthGenerationAlgorithm(config.AuthAlgorithmPlain)
		defer config.SetAuthGenerationAlgorithm(originalAlgorithm)

		account := &VMessAccount{
			NamedAccount: *NewNamedAccount("test", "test-seed"),
		}

		err := validateAndGenerateFields(account, "invalid-uuid-seed")
		if err == nil {
			t.Error("Expected error for invalid UUID in plain mode")
		}
		if !strings.Contains(err.Error(), "failed to generate UUID") {
			t.Errorf("Expected 'failed to generate UUID' error, got %v", err)
		}
	})

	t.Run("Password generation", func(t *testing.T) {
		account := &TrojanAccount{
			NamedAccount: *NewNamedAccount("test", "test-seed"),
		}

		err := validateAndGenerateFields(account, "test-seed")
		if err != nil {
			t.Fatalf("validateAndGenerateFields failed: %v", err)
		}

		if account.Password == "" {
			t.Error("Password should be generated")
		}
	})

	t.Run("Both UUID and Password generation", func(t *testing.T) {
		account := &TUICAccount{
			NamedAccount: *NewNamedAccount("test", "test-seed"),
		}

		err := validateAndGenerateFields(account, "test-seed")
		if err != nil {
			t.Fatalf("validateAndGenerateFields failed: %v", err)
		}

		if account.UUID == "" {
			t.Error("UUID should be generated")
		}

		if account.Password == "" {
			t.Error("Password should be generated")
		}
	})

	t.Run("Preserve existing values", func(t *testing.T) {
		existingUUID := "existing-uuid"
		existingPassword := "existing-password"

		account := &TUICAccount{
			NamedAccount: *NewNamedAccount("test", "test-seed"),
			UUID:         existingUUID,
			Password:     existingPassword,
		}

		err := validateAndGenerateFields(account, "test-seed")
		if err != nil {
			t.Fatalf("validateAndGenerateFields failed: %v", err)
		}

		if account.UUID != existingUUID {
			t.Errorf("UUID should be preserved, expected %s, got %s", existingUUID, account.UUID)
		}

		if account.Password != existingPassword {
			t.Errorf("Password should be preserved, expected %s, got %s", existingPassword, account.Password)
		}
	})

	t.Run("Account without UUID or Password fields", func(t *testing.T) {
		account := &NamedAccount{
			SingboxAccount: SingboxAccount{
				Identifier: "test",
				Seed:       "test-seed",
			},
			Name: "test",
		}

		err := validateAndGenerateFields(account, "test-seed")
		if err != nil {
			t.Fatalf("validateAndGenerateFields should not fail for account without UUID/Password fields: %v", err)
		}
	})

	t.Run("Non-string field types", func(t *testing.T) {
		type testAccount struct {
			SingboxAccount
			NonStringField int
		}

		account := &testAccount{
			SingboxAccount: SingboxAccount{
				Identifier: "test",
				Seed:       "test-seed",
			},
			NonStringField: 42,
		}

		err := validateAndGenerateFields(account, "test-seed")
		if err != nil {
			t.Fatalf("validateAndGenerateFields should handle non-string fields: %v", err)
		}
	})

	t.Run("Non-settable fields", func(t *testing.T) {
		type testAccount struct {
			SingboxAccount
			privateField string
		}

		account := &testAccount{
			SingboxAccount: SingboxAccount{
				Identifier: "test",
				Seed:       "test-seed",
			},
		}

		err := validateAndGenerateFields(account, "test-seed")
		if err != nil {
			t.Fatalf("validateAndGenerateFields should handle non-settable fields: %v", err)
		}
	})
}
