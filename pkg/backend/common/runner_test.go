package common

import (
	"context"
	"os/exec"
	"syscall"
	"testing"
	"time"

	"github.com/highlight-apps/node-backend/logging"
)

type mockController struct {
	isRunning     bool
	stopError     error
	restartError  error
	reloadError   error
	buffer        []string
	onStopHandler func()
	logsChan      chan string
}

func (m *mockController) SetupCmd(cmd *exec.Cmd) error {
	return nil
}

func (m *mockController) IsRunning() bool {
	return m.isRunning
}

func (m *mockController) Stop() error {
	return m.stopError
}

func (m *mockController) Restart(doStart func() error) error {
	if m.restartError != nil {
		return m.restartError
	}
	if doStart != nil {
		return doStart()
	}
	return nil
}

func (m *mockController) Reload(signal syscall.Signal) error {
	return m.reloadError
}

func (m *mockController) GetBuffer() []string {
	return m.buffer
}

func (m *mockController) SubscribeLogs(ctx context.Context) <-chan string {
	if m.logsChan == nil {
		ch := make(chan string)
		close(ch)
		return ch
	}
	return m.logsChan
}

func (m *mockController) SetOnStop(fn func()) {
	m.onStopHandler = fn
}

func TestNewBaseRunner(t *testing.T) {
	logger := logging.NewStdLogger()
	controller := &mockController{}
	execPath := "/path/to/executable"

	runner := NewBaseRunner(execPath, logger, controller)

	if runner.Logger != logger {
		t.Errorf("expected logger to be set")
	}
	if runner.Controller != controller {
		t.Errorf("expected controller to be set")
	}
	if runner.ExecutablePath != execPath {
		t.Errorf("expected executable path to be %s, got %s", execPath, runner.ExecutablePath)
	}
	if runner.stopEvent == nil {
		t.Errorf("expected stopEvent channel to be initialized")
	}
}

func TestBaseRunner_Version_Panics(t *testing.T) {
	runner := createTestBaseRunner()

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected Version to panic")
		}
	}()

	runner.Version()
}

func TestBaseRunner_Start_Panics(t *testing.T) {
	runner := createTestBaseRunner()

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected Start to panic")
		}
	}()

	runner.Start("config")
}

func TestBaseRunner_IsRunning(t *testing.T) {
	tests := []struct {
		name              string
		controllerRunning bool
		expected          bool
	}{
		{
			name:              "Running",
			controllerRunning: true,
			expected:          true,
		},
		{
			name:              "NotRunning",
			controllerRunning: false,
			expected:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := &mockController{isRunning: tt.controllerRunning}
			runner := NewBaseRunner("/test", logging.NewStdLogger(), controller)

			result := runner.IsRunning()
			if result != tt.expected {
				t.Errorf("expected IsRunning to return %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestBaseRunner_Restart_Panics(t *testing.T) {
	runner := createTestBaseRunner()

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected Restart to panic")
		}
	}()

	runner.Restart("config")
}

func TestBaseRunner_Reload_Panics(t *testing.T) {
	runner := createTestBaseRunner()

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected Reload to panic")
		}
	}()

	runner.Reload()
}

func TestBaseRunner_Stop(t *testing.T) {
	tests := []struct {
		name          string
		stopError     error
		expectedError error
	}{
		{
			name:          "Success",
			stopError:     nil,
			expectedError: nil,
		},
		{
			name:          "Error",
			stopError:     ErrFailedStopRunner,
			expectedError: ErrFailedStopRunner,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := &mockController{stopError: tt.stopError}
			runner := NewBaseRunner("/test", logging.NewStdLogger(), controller)

			err := runner.Stop()
			if err != tt.expectedError {
				t.Errorf("expected error %v, got %v", tt.expectedError, err)
			}
		})
	}
}

func TestBaseRunner_SubscribeLogs(t *testing.T) {
	logsChan := make(chan string, 1)
	logsChan <- "test log"
	close(logsChan)

	controller := &mockController{logsChan: logsChan}
	runner := NewBaseRunner("/test", logging.NewStdLogger(), controller)

	ctx := context.Background()
	resultChan := runner.SubscribeLogs(ctx)

	log, ok := <-resultChan
	if !ok {
		t.Fatal("expected to receive log from channel")
	}
	if log != "test log" {
		t.Errorf("expected log 'test log', got %s", log)
	}
}

func TestBaseRunner_GetBuffer(t *testing.T) {
	expectedBuffer := []string{"log1", "log2", "log3"}
	controller := &mockController{buffer: expectedBuffer}
	runner := NewBaseRunner("/test", logging.NewStdLogger(), controller)

	buffer := runner.GetBuffer()
	if len(buffer) != len(expectedBuffer) {
		t.Errorf("expected buffer length %d, got %d", len(expectedBuffer), len(buffer))
	}
	for i, log := range buffer {
		if log != expectedBuffer[i] {
			t.Errorf("expected buffer[%d] to be %s, got %s", i, expectedBuffer[i], log)
		}
	}
}

func TestBaseRunner_TriggerStopEvent(t *testing.T) {
	runner := createTestBaseRunner()

	stopEventCh := runner.StopEvent()

	select {
	case <-stopEventCh:
		t.Fatal("expected stop event channel to be open initially")
	default:
	}

	runner.TriggerStopEvent()

	select {
	case <-stopEventCh:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected stop event channel to be closed after trigger")
	}
}

func TestBaseRunner_TriggerStopEvent_MultipleCallsIdempotent(t *testing.T) {
	runner := createTestBaseRunner()

	stopEventCh := runner.StopEvent()

	runner.TriggerStopEvent()
	runner.TriggerStopEvent()
	runner.TriggerStopEvent()

	select {
	case <-stopEventCh:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected stop event channel to be closed")
	}
}

func TestBaseRunner_ResetStopEvent(t *testing.T) {
	runner := createTestBaseRunner()

	firstCh := runner.StopEvent()

	runner.TriggerStopEvent()

	select {
	case <-firstCh:
	default:
		t.Fatal("expected first channel to be closed")
	}

	runner.ResetStopEvent()

	secondCh := runner.StopEvent()

	if firstCh == secondCh {
		t.Error("expected new channel after reset")
	}

	select {
	case <-secondCh:
		t.Fatal("expected new channel to be open")
	default:
	}
}

func TestBaseRunner_StopEvent(t *testing.T) {
	runner := createTestBaseRunner()

	ch1 := runner.StopEvent()
	ch2 := runner.StopEvent()

	if ch1 != ch2 {
		t.Error("expected StopEvent to return same channel on multiple calls")
	}
}

func TestBaseRunner_SetupOnStopHandler(t *testing.T) {
	controller := &mockController{}
	runner := NewBaseRunner("/test", logging.NewStdLogger(), controller)

	runner.SetupOnStopHandler()

	if controller.onStopHandler == nil {
		t.Fatal("expected onStopHandler to be set")
	}

	stopEventCh := runner.StopEvent()

	controller.onStopHandler()

	select {
	case <-stopEventCh:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected stop event to be triggered by handler")
	}
}

func TestBaseRunner_RestartWithCallback(t *testing.T) {
	tests := []struct {
		name          string
		restartError  error
		expectedError error
	}{
		{
			name:          "Success",
			restartError:  nil,
			expectedError: nil,
		},
		{
			name:          "Error",
			restartError:  ErrProcessAlreadyRestarting,
			expectedError: ErrProcessAlreadyRestarting,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := &mockController{restartError: tt.restartError}
			runner := NewBaseRunner("/test", logging.NewStdLogger(), controller)

			callbackCalled := false
			callback := func() error {
				callbackCalled = true
				return nil
			}

			err := runner.RestartWithCallback(callback)

			if err != tt.expectedError {
				t.Errorf("expected error %v, got %v", tt.expectedError, err)
			}

			if tt.expectedError == nil && !callbackCalled {
				t.Error("expected callback to be called on successful restart")
			}
		})
	}
}

func createTestBaseRunner() *BaseRunner {
	return NewBaseRunner("/test/path", logging.NewStdLogger(), &mockController{})
}
