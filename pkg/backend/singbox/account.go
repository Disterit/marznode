package singbox

import (
	"fmt"
	"reflect"

	"github.com/highlight-apps/node-backend/utils"
)

type XTLSFlow string

const (
	XTLSFlowNone   XTLSFlow = ""
	XTLSFlowVision XTLSFlow = "xtls-rprx-vision"
)

type Account interface {
	GetIdentifier() string
	GetSeed() string
	ToDict() map[string]any
	String() string
}

type SingboxAccount struct {
	Identifier string `json:"-"`
	Seed       string `json:"-"`
}

func (a *SingboxAccount) GetIdentifier() string {
	return a.Identifier
}

func (a *SingboxAccount) GetSeed() string {
	return a.Seed
}

func (a *SingboxAccount) String() string {
	return fmt.Sprintf("<%T %s>", a, a.Identifier)
}

type NamedAccount struct {
	SingboxAccount
	Name string `json:"name"`
}

func NewNamedAccount(identifier, seed string) *NamedAccount {
	return &NamedAccount{
		SingboxAccount: SingboxAccount{Identifier: identifier, Seed: seed},
		Name:           identifier,
	}
}

func (a *NamedAccount) ToDict() map[string]any {
	return map[string]any{
		"name": a.Name,
	}
}

type UserNamedAccount struct {
	SingboxAccount
	Username string `json:"username"`
}

func NewUserNamedAccount(identifier, seed string) *UserNamedAccount {
	return &UserNamedAccount{
		SingboxAccount: SingboxAccount{Identifier: identifier, Seed: seed},
		Username:       identifier,
	}
}

func (a *UserNamedAccount) ToDict() map[string]any {
	return map[string]any{
		"username": a.Username,
	}
}

type VMessAccount struct {
	NamedAccount
	UUID string `json:"uuid"`
}

func NewVMessAccount(identifier, seed string, uuid string) (*VMessAccount, error) {
	account := &VMessAccount{
		NamedAccount: *NewNamedAccount(identifier, seed),
	}

	if uuid != "" {
		account.UUID = uuid
	}

	if err := validateAndGenerateFields(account, seed); err != nil {
		return nil, err
	}

	return account, nil
}

func (a *VMessAccount) ToDict() map[string]any {
	return map[string]any{
		"name": a.Name,
		"uuid": a.UUID,
	}
}

type VLESSAccount struct {
	NamedAccount
	UUID string   `json:"uuid"`
	Flow XTLSFlow `json:"flow"`
}

func NewVLESSAccount(identifier, seed string, uuid string, flow XTLSFlow) (*VLESSAccount, error) {
	account := &VLESSAccount{
		NamedAccount: *NewNamedAccount(identifier, seed),
		Flow:         flow,
	}

	if uuid != "" {
		account.UUID = uuid
	}

	if err := validateAndGenerateFields(account, seed); err != nil {
		return nil, err
	}

	return account, nil
}

func (a *VLESSAccount) ToDict() map[string]any {
	return map[string]any{
		"name": a.Name,
		"uuid": a.UUID,
		"flow": a.Flow,
	}
}

type TrojanAccount struct {
	NamedAccount
	Password string `json:"password"`
}

func NewTrojanAccount(identifier, seed string, password string) (*TrojanAccount, error) {
	account := &TrojanAccount{
		NamedAccount: *NewNamedAccount(identifier, seed),
	}

	if password != "" {
		account.Password = password
	}

	if err := validateAndGenerateFields(account, seed); err != nil {
		return nil, err
	}

	return account, nil
}

func (a *TrojanAccount) ToDict() map[string]any {
	return map[string]any{
		"name":     a.Name,
		"password": a.Password,
	}
}

type ShadowsocksAccount struct {
	NamedAccount
	Password string `json:"password"`
}

func NewShadowsocksAccount(identifier, seed string, password string) (*ShadowsocksAccount, error) {
	account := &ShadowsocksAccount{
		NamedAccount: *NewNamedAccount(identifier, seed),
	}

	if password != "" {
		account.Password = password
	}

	if err := validateAndGenerateFields(account, seed); err != nil {
		return nil, err
	}

	return account, nil
}

func (a *ShadowsocksAccount) ToDict() map[string]any {
	return map[string]any{
		"name":     a.Name,
		"password": a.Password,
	}
}

type TUICAccount struct {
	NamedAccount
	UUID     string `json:"uuid"`
	Password string `json:"password"`
}

func NewTUICAccount(identifier, seed, uuid, password string) (*TUICAccount, error) {
	account := &TUICAccount{
		NamedAccount: *NewNamedAccount(identifier, seed),
	}

	if uuid != "" {
		account.UUID = uuid
	}
	if password != "" {
		account.Password = password
	}

	if err := validateAndGenerateFields(account, seed); err != nil {
		return nil, err
	}

	return account, nil
}

func (a *TUICAccount) ToDict() map[string]any {
	return map[string]any{
		"name":     a.Name,
		"uuid":     a.UUID,
		"password": a.Password,
	}
}

type Hysteria2Account struct {
	NamedAccount
	Password string `json:"password"`
}

func NewHysteria2Account(identifier, seed, password string) (*Hysteria2Account, error) {
	account := &Hysteria2Account{
		NamedAccount: *NewNamedAccount(identifier, seed),
	}

	if password != "" {
		account.Password = password
	}

	if err := validateAndGenerateFields(account, seed); err != nil {
		return nil, err
	}

	return account, nil
}

func (a *Hysteria2Account) ToDict() map[string]any {
	return map[string]any{
		"name":     a.Name,
		"password": a.Password,
	}
}

type NaiveAccount struct {
	UserNamedAccount
	Password string `json:"password"`
}

func NewNaiveAccount(identifier, seed, password string) (*NaiveAccount, error) {
	account := &NaiveAccount{
		UserNamedAccount: *NewUserNamedAccount(identifier, seed),
	}

	if password != "" {
		account.Password = password
	}

	if err := validateAndGenerateFields(account, seed); err != nil {
		return nil, err
	}

	return account, nil
}

func (a *NaiveAccount) ToDict() map[string]any {
	return map[string]any{
		"username": a.Username,
		"password": a.Password,
	}
}

type ShadowTLSAccount struct {
	NamedAccount
	Password string `json:"password"`
}

func NewShadowTLSAccount(identifier, seed, password string) (*ShadowTLSAccount, error) {
	account := &ShadowTLSAccount{
		NamedAccount: *NewNamedAccount(identifier, seed),
	}

	if password != "" {
		account.Password = password
	}

	if err := validateAndGenerateFields(account, seed); err != nil {
		return nil, err
	}

	return account, nil
}

func (a *ShadowTLSAccount) ToDict() map[string]any {
	return map[string]any{
		"name":     a.Name,
		"password": a.Password,
	}
}

type SocksAccount struct {
	UserNamedAccount
	Password string `json:"password"`
}

func NewSocksAccount(identifier, seed, password string) (*SocksAccount, error) {
	account := &SocksAccount{
		UserNamedAccount: *NewUserNamedAccount(identifier, seed),
	}

	if password != "" {
		account.Password = password
	}

	if err := validateAndGenerateFields(account, seed); err != nil {
		return nil, err
	}

	return account, nil
}

func (a *SocksAccount) ToDict() map[string]any {
	return map[string]any{
		"username": a.Username,
		"password": a.Password,
	}
}

type HTTPAccount struct {
	UserNamedAccount
	Password string `json:"password"`
}

func NewHTTPAccount(identifier, seed, password string) (*HTTPAccount, error) {
	account := &HTTPAccount{
		UserNamedAccount: *NewUserNamedAccount(identifier, seed),
	}

	if password != "" {
		account.Password = password
	}

	if err := validateAndGenerateFields(account, seed); err != nil {
		return nil, err
	}

	return account, nil
}

func (a *HTTPAccount) ToDict() map[string]any {
	return map[string]any{
		"username": a.Username,
		"password": a.Password,
	}
}

type MixedAccount struct {
	UserNamedAccount
	Password string `json:"password"`
}

func NewMixedAccount(identifier, seed, password string) (*MixedAccount, error) {
	account := &MixedAccount{
		UserNamedAccount: *NewUserNamedAccount(identifier, seed),
	}

	if password != "" {
		account.Password = password
	}

	if err := validateAndGenerateFields(account, seed); err != nil {
		return nil, err
	}

	return account, nil
}

func (a *MixedAccount) ToDict() map[string]any {
	return map[string]any{
		"username": a.Username,
		"password": a.Password,
	}
}

type AccountOptions struct {
	UUID     string
	Password string
	Flow     XTLSFlow
}

type AccountFactory func(identifier, seed string, opts *AccountOptions) (Account, error)

var AccountsMap = map[string]AccountFactory{
	"shadowsocks": func(identifier, seed string, opts *AccountOptions) (Account, error) {
		password := ""
		if opts != nil && opts.Password != "" {
			password = opts.Password
		}
		return NewShadowsocksAccount(identifier, seed, password)
	},
	"trojan": func(identifier, seed string, opts *AccountOptions) (Account, error) {
		password := ""
		if opts != nil && opts.Password != "" {
			password = opts.Password
		}
		return NewTrojanAccount(identifier, seed, password)
	},
	"vmess": func(identifier, seed string, opts *AccountOptions) (Account, error) {
		uuid := ""
		if opts != nil && opts.UUID != "" {
			uuid = opts.UUID
		}
		return NewVMessAccount(identifier, seed, uuid)
	},
	"vless": func(identifier, seed string, opts *AccountOptions) (Account, error) {
		uuid := ""
		flow := XTLSFlowNone
		if opts != nil {
			if opts.UUID != "" {
				uuid = opts.UUID
			}
			if opts.Flow != "" {
				flow = opts.Flow
			}
		}
		return NewVLESSAccount(identifier, seed, uuid, flow)
	},
	"tuic": func(identifier, seed string, opts *AccountOptions) (Account, error) {
		uuid := ""
		password := ""
		if opts != nil {
			if opts.UUID != "" {
				uuid = opts.UUID
			}
			if opts.Password != "" {
				password = opts.Password
			}
		}
		return NewTUICAccount(identifier, seed, uuid, password)
	},
	"shadowtls": func(identifier, seed string, opts *AccountOptions) (Account, error) {
		password := ""
		if opts != nil && opts.Password != "" {
			password = opts.Password
		}
		return NewShadowTLSAccount(identifier, seed, password)
	},
	"hysteria2": func(identifier, seed string, opts *AccountOptions) (Account, error) {
		password := ""
		if opts != nil && opts.Password != "" {
			password = opts.Password
		}
		return NewHysteria2Account(identifier, seed, password)
	},
	"naive": func(identifier, seed string, opts *AccountOptions) (Account, error) {
		password := ""
		if opts != nil && opts.Password != "" {
			password = opts.Password
		}
		return NewNaiveAccount(identifier, seed, password)
	},
	"socks": func(identifier, seed string, opts *AccountOptions) (Account, error) {
		password := ""
		if opts != nil && opts.Password != "" {
			password = opts.Password
		}
		return NewSocksAccount(identifier, seed, password)
	},
	"mixed": func(identifier, seed string, opts *AccountOptions) (Account, error) {
		password := ""
		if opts != nil && opts.Password != "" {
			password = opts.Password
		}
		return NewMixedAccount(identifier, seed, password)
	},
	"http": func(identifier, seed string, opts *AccountOptions) (Account, error) {
		password := ""
		if opts != nil && opts.Password != "" {
			password = opts.Password
		}
		return NewHTTPAccount(identifier, seed, password)
	},
}

func CreateAccount(protocol, identifier, seed string, opts *AccountOptions) (Account, error) {
	factory, exists := AccountsMap[protocol]
	if !exists {
		return nil, fmt.Errorf("unsupported protocol: %s", protocol)
	}
	return factory(identifier, seed, opts)
}

func validateAndGenerateFields(account interface{}, seed string) error {
	if seed == "" {
		return fmt.Errorf("seed cannot be empty")
	}

	v := reflect.ValueOf(account).Elem()
	t := reflect.TypeOf(account).Elem()

	for i := range v.NumField() {
		field := v.Field(i)
		fieldType := t.Field(i)
		fieldName := fieldType.Name

		if !field.CanSet() {
			continue
		}

		switch fieldName {
		case "UUID":
			if field.Kind() == reflect.String {
				currentValue := field.String()
				if currentValue == "" {
					uuid, err := utils.GenerateUUIDString(seed)
					if err != nil {
						return fmt.Errorf("failed to generate UUID: %w", err)
					}
					field.SetString(uuid)
				}
			}
		case "Password":
			if field.Kind() == reflect.String {
				currentValue := field.String()
				if currentValue == "" {
					password := utils.GeneratePassword(seed)
					field.SetString(password)
				}
			}
		}
	}

	return nil
}
