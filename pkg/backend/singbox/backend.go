package singbox

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/highlight-apps/node-backend/backend/common"
	"github.com/highlight-apps/node-backend/backend/common/models"
	"github.com/highlight-apps/node-backend/config"
	"github.com/highlight-apps/node-backend/logging"
	"github.com/highlight-apps/node-backend/storage"
	"github.com/highlight-apps/node-backend/utils"
)

var _ common.VPNBackend = (*SingBoxBackend)(nil)

type SingBoxBackend struct {
	config                  *SingBoxConfig
	configUpdateEvent       chan struct{}
	inboundTags             map[string]bool
	inbounds                []models.Inbound
	api                     *SingBoxAPI
	runner                  *SingboxRunner
	storage                 storage.BaseStorage
	configPath              string
	fullConfigPath          string
	restartMutex            sync.Mutex
	configModificationMutex sync.Mutex
	logger                  logging.Logger
}

func NewSingBoxBackend(executablePath, configPath string, store storage.BaseStorage, logger logging.Logger) (*SingBoxBackend, error) {
	runner, err := NewSingboxRunner(executablePath, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create runner: %w", err)
	}

	backend := &SingBoxBackend{
		configUpdateEvent: make(chan struct{}, 1),
		inboundTags:       make(map[string]bool),
		inbounds:          make([]models.Inbound, 0),
		runner:            runner,
		storage:           store,
		configPath:        configPath,
		fullConfigPath:    configPath + ".full",
		logger:            logger,
	}

	go backend.restartOnFailure()
	go backend.userUpdateHandler()

	return backend, nil
}

func (s *SingBoxBackend) BackendType() string {
	return "sing-box"
}

func (s *SingBoxBackend) ConfigFormat() int {
	return 1
}

func (s *SingBoxBackend) Version() (string, error) {
	return s.runner.Version()
}

func (s *SingBoxBackend) Running() bool {
	return s.runner.IsRunning()
}

func (s *SingBoxBackend) ContainsTag(tag string) bool {
	return s.inboundTags[tag]
}

func (s *SingBoxBackend) userUpdateHandler() {
	interval := time.Duration(config.SingBoxUserModificationInterval) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.logger.Debug("checking for sing-box user modifications")
			s.configModificationMutex.Lock()
			select {
			case <-s.configUpdateEvent:
				s.logger.Debug("updating sing-box users")
				configJSON, err := s.config.ToJSON()
				if err != nil {
					s.logger.Error("failed to convert config to JSON:", err)
					s.configModificationMutex.Unlock()
					continue
				}
				s.saveConfig(configJSON, true)
				if err := s.runner.Reload(); err != nil {
					s.logger.Error("failed to reload runner:", err)
				}
			default:
			}
			s.configModificationMutex.Unlock()
		}
	}
}

func (s *SingBoxBackend) ListInbounds(ctx context.Context) ([]models.Inbound, error) {
	return s.inbounds, nil
}

func (s *SingBoxBackend) GetConfig(ctx context.Context) (any, error) {
	data, err := os.ReadFile(s.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}
	return string(data), nil
}

func (s *SingBoxBackend) saveConfig(config string, full bool) error {
	path := s.configPath
	if full {
		path = s.fullConfigPath
	}

	return os.WriteFile(path, []byte(config), 0644)
}

func (s *SingBoxBackend) addStorageUsers(ctx context.Context) error {
	for _, inbound := range s.inbounds {
		users, err := s.storage.ListInboundUsers(inbound.Tag)
		if err != nil {
			return fmt.Errorf("failed to list users for inbound %s: %w", inbound.Tag, err)
		}
		for _, user := range users {
			if err := s.config.AppendUser(user, inbound); err != nil {
				return fmt.Errorf("failed to append user %s to inbound %s: %w", user.Username, inbound.Tag, err)
			}
		}
	}
	return nil
}

func (s *SingBoxBackend) restartOnFailure() {
	stopEvent := s.runner.StopEvent()
	for {
		<-stopEvent
		if s.restartMutex.TryLock() {
			s.logger.Debug("Sing-box stopped unexpectedly")
			s.restartMutex.Unlock()
			if config.SingBoxRestartOnFailure {
				restartInterval := time.Duration(config.SingBoxRestartOnFailureInterval) * time.Second
				time.Sleep(restartInterval)

				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				if err := s.Start(ctx, nil); err != nil {
					s.logger.Error("failed to restart sing-box:", err)
				}
				cancel()
			}
		} else {
			s.logger.Debug("Sing-box restarting as planned")
		}
		stopEvent = s.runner.StopEvent()
	}
}

func (s *SingBoxBackend) Start(ctx context.Context, backendConfig any) error {
	var configStr string

	if backendConfig == nil {
		data, err := os.ReadFile(s.configPath)
		if err != nil {
			return fmt.Errorf("failed to read config: %w", err)
		}
		configStr = string(data)
	} else {
		switch cfg := backendConfig.(type) {
		case string:
			configStr = cfg
		case []byte:
			configStr = string(cfg)
		default:
			data, err := json.Marshal(cfg)
			if err != nil {
				return fmt.Errorf("failed to marshal config: %w", err)
			}
			configStr = string(data)
		}

		var prettyConfig map[string]any
		if err := json.Unmarshal([]byte(configStr), &prettyConfig); err != nil {
			return fmt.Errorf("failed to parse config JSON: %w", err)
		}

		prettyData, err := json.MarshalIndent(prettyConfig, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to format config: %w", err)
		}

		if err := s.saveConfig(string(prettyData), false); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		configStr = string(prettyData)
	}

	config, err := NewSingBoxConfig(configStr, "127.0.0.1", 8081)
	if err != nil {
		return fmt.Errorf("failed to create config: %w", err)
	}

	s.config = config
	s.inboundTags = make(map[string]bool)
	s.inbounds = config.ListInbounds()

	for _, inbound := range s.inbounds {
		s.inboundTags[inbound.Tag] = true
	}

	if err := s.config.RegisterInbounds(s.storage); err != nil {
		return fmt.Errorf("failed to register inbounds: %w", err)
	}

	if err := s.addStorageUsers(ctx); err != nil {
		return fmt.Errorf("failed to add storage users: %w", err)
	}

	configJSON, err := s.config.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to convert config to JSON: %w", err)
	}

	if err := s.saveConfig(configJSON, true); err != nil {
		return fmt.Errorf("failed to save full config: %w", err)
	}

	api, err := NewSingBoxAPI("127.0.0.1", config.ApiPort)
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}
	s.api = api

	return s.runner.Start(configJSON)
}

func (s *SingBoxBackend) stop(ctx context.Context) error {
	if err := s.runner.Stop(); err != nil {
		return fmt.Errorf("failed to stop runner: %w", err)
	}

	for _, inbound := range s.inbounds {
		if err := s.storage.RemoveInbound(inbound); err != nil {
			s.logger.Error("failed to remove inbound:", inbound.Tag, err)
		}
	}

	s.inboundTags = make(map[string]bool)
	s.inbounds = make([]models.Inbound, 0)

	return nil
}

func (s *SingBoxBackend) Restart(ctx context.Context, backendConfig any) error {
	s.restartMutex.Lock()
	defer s.restartMutex.Unlock()

	if isEmpty(backendConfig) {
		configJSON, err := s.config.ToJSON()
		if err != nil {
			return fmt.Errorf("failed to get current config: %w", err)
		}
		return s.runner.Restart(configJSON)
	}

	if err := s.stop(ctx); err != nil {
		return fmt.Errorf("failed to stop during restart: %w", err)
	}

	return s.Start(ctx, backendConfig)
}

func (s *SingBoxBackend) AddUser(ctx context.Context, user models.User, inbound models.Inbound) error {
	s.configModificationMutex.Lock()
	defer s.configModificationMutex.Unlock()

	if err := s.config.AppendUser(user, inbound); err != nil {
		return fmt.Errorf("failed to append user: %w", err)
	}

	select {
	case s.configUpdateEvent <- struct{}{}:
	default:
	}

	return nil
}

func (s *SingBoxBackend) RemoveUser(ctx context.Context, user models.User, inbound models.Inbound) error {
	s.configModificationMutex.Lock()
	defer s.configModificationMutex.Unlock()

	if err := s.config.PopUser(user, inbound); err != nil {
		return fmt.Errorf("failed to remove user: %w", err)
	}

	select {
	case s.configUpdateEvent <- struct{}{}:
	default:
	}

	return nil
}

func (s *SingBoxBackend) GetLogs(ctx context.Context, includeBuffer bool) (<-chan string, error) {
	logChan := make(chan string, 100)

	go func() {
		defer close(logChan)

		if includeBuffer {
			buffer := s.runner.GetBuffer()
			for _, line := range buffer {
				select {
				case logChan <- line:
				case <-ctx.Done():
					return
				}
			}
		}

		streamChan := s.runner.SubscribeLogs(ctx)
		for line := range streamChan {
			select {
			case logChan <- line:
			case <-ctx.Done():
				return
			}
		}
	}()

	return logChan, nil
}

func (s *SingBoxBackend) GetUsages(ctx context.Context) (any, error) {
	if s.api == nil {
		return make(map[int64]int64), nil
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	apiStats, err := s.api.GetUsersStats(timeoutCtx, true)
	if err != nil {
		s.logger.Error("failed to get stats:", err)
		return make(map[int64]int64), nil
	}

	stats := make(map[int64]int64)
	for _, stat := range apiStats {
		parts := strings.Split(stat.Name, ".")
		if len(parts) > 0 {
			if uid, err := strconv.ParseInt(parts[0], 10, 64); err == nil {
				stats[uid] += int64(stat.Value)
			}
		}
	}

	return stats, nil
}

func isEmpty(value any) bool {
	if value == nil {
		return true
	}

	if str, ok := value.(string); ok && str == "" {
		return true
	}

	return false
}
