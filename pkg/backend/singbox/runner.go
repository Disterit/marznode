package singbox

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/highlight-apps/node-backend/backend/common"
	"github.com/highlight-apps/node-backend/logging"
)

var execCommand = exec.Command
var osCreateTemp = os.CreateTemp
var fileCloser func(*os.File) error = (*os.File).Close

type SingboxRunner struct {
	*common.BaseRunner
	assetsPath     string
	configFilePath string
}

var _ common.Runner = (*SingboxRunner)(nil)

func NewSingboxRunner(executablePath string, logger logging.Logger) (*SingboxRunner, error) {
	var l logging.Logger
	if logger != nil {
		l = logger
	} else {
		l = logging.NewStdLogger()
	}

	pc := common.NewProcessController(l)
	Runner := common.NewBaseRunner(executablePath, l, pc)
	r := &SingboxRunner{
		BaseRunner: Runner,
	}
	return r, nil
}

func (r *SingboxRunner) Version() (string, error) {
	version, err := getSingboxVersion(r.ExecutablePath)
	if err != nil {
		return "", fmt.Errorf("%w: %v", common.ErrFailedToGetVersion, err)
	}
	return version, nil
}

func (r *SingboxRunner) Start(config string) error {
	if r.Controller.IsRunning() {
		r.Logger.Error("sing-box is started already")
		return common.ErrProcessAlreadyRunning
	}

	configPath, err := r.createConfigFile(config)
	if err != nil {
		r.Logger.Error("failed to create config file:", err)
		return fmt.Errorf("failed to create config file: %w", err)
	}

	cmd := execCommand(r.ExecutablePath, "run", "--disable-color", "-c", configPath)

	r.SetupOnStopHandler()

	if err := r.Controller.SetupCmd(cmd); err != nil {
		r.Logger.Error("failed to start sing-box:", err)
		r.removeConfigFile()
		return fmt.Errorf("failed to start sing-box: %w", err)
	}

	r.Logger.Info("sing-box started")
	return nil
}

func (r *SingboxRunner) Restart(config string) error {
	r.Logger.Info("Restarting sing-box")

	return r.Controller.Restart(func() error {
		r.ResetStopEvent()
		return r.Start(config)
	})
}

func (r *SingboxRunner) Reload() error {
	return r.Controller.Reload(syscall.SIGHUP)
}

func (r *SingboxRunner) Stop() error {
	if !r.IsRunning() {
		return nil
	}

	if err := r.Controller.Stop(); err != nil {
		return err
	}

	r.removeConfigFile()
	r.TriggerStopEvent()

	r.Logger.Info("sing-box stopped")
	return nil
}

func (r *SingboxRunner) createConfigFile(config string) (string, error) {
	if r.configFilePath != "" {
		os.Remove(r.configFilePath)
	}

	tmpDir := os.TempDir()
	configFile, err := osCreateTemp(tmpDir, "singbox-config-*.json")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary config file: %w", err)
	}

	if _, err := configFile.WriteString(config); err != nil {
		configFile.Close()
		os.Remove(configFile.Name())
		return "", fmt.Errorf("failed to write config to file: %w", err)
	}

	if err := fileCloser(configFile); err != nil {
		os.Remove(configFile.Name())
		return "", fmt.Errorf("failed to close config file: %w", err)
	}

	r.configFilePath = configFile.Name()
	r.Logger.Info("Created temporary config file:", r.configFilePath)

	return r.configFilePath, nil
}

func (r *SingboxRunner) removeConfigFile() {
	if r.configFilePath != "" {
		if err := os.Remove(r.configFilePath); err != nil {
			r.Logger.Error("failed to remove config file:", err)
		} else {
			r.Logger.Info("Removed temporary config file:", r.configFilePath)
		}
		r.configFilePath = ""
	}
}
