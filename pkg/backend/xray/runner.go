package xray

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"time"

	"github.com/highlight-apps/node-backend/backend/common"
	"github.com/highlight-apps/node-backend/logging"
)

var _ common.Runner = (*XrayRunner)(nil)

var (
	execCommand    = exec.Command
	newStdinPipe   = func(cmd *exec.Cmd) (io.WriteCloser, error) { return cmd.StdinPipe() }
	startupTimeout = 4 * time.Second
)

type XrayRunner struct {
	*common.BaseRunner
	assetsPath string
}

func NewXrayRunner(executablePath string, assetsPath string, logger ...logging.Logger) (*XrayRunner, error) {
	var l logging.Logger
	if len(logger) > 0 {
		l = logger[0]
	} else {
		l = logging.NewStdLogger()
	}
	controller := common.NewProcessController(l)
	baseRunner := common.NewBaseRunner(executablePath, l, controller)
	runner := &XrayRunner{
		BaseRunner: baseRunner,
		assetsPath: assetsPath,
	}
	l.Info("Xray runner initialized")
	return runner, nil
}

func (r *XrayRunner) Version() (string, error) {
	version, err := getXrayVersion(r.ExecutablePath)
	if err != nil {
		return "", fmt.Errorf("%w: %v", common.ErrFailedToGetVersion, err)
	}
	return version, nil
}

func (r *XrayRunner) Start(config string) error {
	if r.Controller.IsRunning() {
		r.Logger.Error("Xray runner already running")
		return common.ErrProcessAlreadyRunning
	}

	cmd := execCommand(r.ExecutablePath, "run", "-config", "stdin:", "--location-asset", r.assetsPath)
	cmd.Env = append(os.Environ(), "XRAY_LOCATION_ASSET="+r.assetsPath)

	stdinPipe, err := newStdinPipe(cmd)
	if err != nil {
		r.Logger.Error("failed to get xray stdin pipe:", err)
		return fmt.Errorf("%w: %v", common.ErrFailedToStartRunner, err)
	}

	r.SetupOnStopHandler()

	if err := r.Controller.SetupCmd(cmd); err != nil {
		r.Logger.Error("failed to start xray:", err)
		return fmt.Errorf("%w: %v", common.ErrFailedToStartRunner, err)
	}

	go func() {
		defer stdinPipe.Close()
		if _, err := stdinPipe.Write([]byte(config)); err != nil {
			r.Logger.Error("failed to write xray config to stdin:", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), startupTimeout)
	defer cancel()

	logsCh := r.Controller.SubscribeLogs(ctx)

	rx := regexp.MustCompile(`\[Warning\] core: Xray \d+\.\d+\.\d+ started`)

	for {
		select {
		case line, ok := <-logsCh:
			if !ok {
				return fmt.Errorf("%w: xray process stopped unexpectedly", common.ErrFailedToStartRunner)
			}

			if rx.MatchString(line) {
				r.Logger.Info("Xray runner started")
				return nil
			}

		case <-ctx.Done():
			return fmt.Errorf("%w: startup log not found within timeout", common.ErrFailedToStartRunner)
		}
	}
}

func (r *XrayRunner) Restart(config string) error {
	return r.Controller.Restart(func() error {
		r.ResetStopEvent()
		r.Logger.Info("Xray runner restarted")
		return r.Start(config)
	})
}

func (r *XrayRunner) Reload() error {
	// Xray doesn't support reload on the fly
	return nil
}

func (r *XrayRunner) Stop() error {
	if err := r.Controller.Stop(); err != nil {
		return err
	}
	r.TriggerStopEvent()
	r.Logger.Info("Xray runner stopped")
	return nil
}
