package common

import (
	"errors"
)

var (
	ErrProcessAlreadyRunning    = errors.New("runner is already running")
	ErrUnknownConfigType        = errors.New("unknown config type")
	ErrFailedReadEmbeddedConfig = errors.New("failed to read embedded config")
	ErrConfigNotSet             = errors.New("config not set")
	ErrFailedWriteConfig        = errors.New("failed to write singbox config")
	ErrFailedCreateTempDir      = errors.New("failed to create temp file for singbox config")
	ErrFailedCloseTempFile      = errors.New("failed to close temp file")
	ErrFailedToStartRunner      = errors.New("failed to start runner")
	ErrProcessNotRunning        = errors.New("process is not running")
	ErrProcessAlreadyRestarting = errors.New("process is already restarting")
	ErrFailedStopRunner         = errors.New("failed to stop runner")
	ErrFailedToGetVersion       = errors.New("failed to get version")
	ErrFailedToParseVersion     = errors.New("failed to parse version")
)
