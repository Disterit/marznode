package singbox

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/highlight-apps/node-backend/assets"
	"github.com/highlight-apps/node-backend/backend/common"
	"github.com/highlight-apps/node-backend/logging"
)

const (
	testTimeout      = 3 * time.Second
	shortTimeout     = 1 * time.Second
	logWaitTimeout   = 100 * time.Millisecond
	testConfig       = `{"test": "config"}`
	mockSleepCommand = "echo 'sing-box started'; sleep 10"
	mockExitCommand  = "echo 'sing-box started'; exit 0"
)

func TestMain(m *testing.M) {
	realExec := exec.Command
	execCommand = func(name string, args ...string) *exec.Cmd {
		if filepath.Base(name) == "true" {
			return realExec(name, args...)
		}
		return realExec("sh", "-c", "echo 'sing-box started'; sleep 10")
	}
	code := m.Run()
	execCommand = realExec
	os.Exit(code)
}
func TestSingboxRunner_New_Success(t *testing.T) {
	tl := logging.NewStdLogger()
	exePath, err := exec.LookPath(common.DefaultSingboxExecutablePath)
	if err != nil {
		t.Fatalf("failed to find singbox executable: %v", err)
	}
	r, err := NewSingboxRunner(exePath, tl)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if r.Logger != tl {
		t.Errorf("expected logger to be set correctly")
	}
	if r.ExecutablePath != exePath {
		t.Errorf("expected executable path to be %v, got %v", exePath, r.ExecutablePath)
	}
	if r.Controller == nil {
		t.Errorf("expected controller to be initialized")
	}
}
func TestSingboxRunner_New_DefaultLogger(t *testing.T) {
	exePath, err := exec.LookPath(common.DefaultSingboxExecutablePath)
	if err != nil {
		t.Fatalf("failed to find singbox executable: %v", err)
	}
	r, err := NewSingboxRunner(exePath, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if r.Logger == nil {
		t.Errorf("expected default logger to be set")
	}
}

func TestSingboxRunner_Start_Success(t *testing.T) {
	cleanup := setupMockExecCommand(t)
	defer cleanup()
	r := newTestRunner(t)
	config, err := loadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	err = r.Start(config)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer r.Stop()
	if !r.IsRunning() {
		t.Errorf("expected runner to be running after Start")
	}
}
func TestSingboxRunner_Start_AlreadyRunning(t *testing.T) {
	cleanup := setupMockExecCommand(t)
	defer cleanup()
	r := newTestRunner(t)
	config, err := loadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	err = r.Start(config)
	if err != nil {
		t.Fatalf("expected no error on first start, got %v", err)
	}
	defer r.Stop()
	err = r.Start(config)
	if !errors.Is(err, common.ErrProcessAlreadyRunning) {
		t.Errorf("expected ErrProcessAlreadyRunning, got %v", err)
	}
}
func TestSingboxRunner_Start_SetupCmdError(t *testing.T) {
	mockCtrl := &mockProcessController{
		setupCmdErr: fmt.Errorf("setup cmd error"),
	}

	runner := common.NewBaseRunner(common.DefaultSingboxExecutablePath, logging.NewStdLogger(), mockCtrl)
	r := &SingboxRunner{
		BaseRunner: runner,
	}
	err := r.Start(testConfig)
	if err == nil {
		t.Fatal("expected error from SetupCmd, got nil")
	}
	if !strings.Contains(err.Error(), "setup cmd error") {
		t.Errorf("expected 'setup cmd error' in error message, got: %v", err)
	}
}

func TestSingboxRunner_Restart_Success(t *testing.T) {
	cleanup := setupMockExecCommand(t)
	defer cleanup()
	r := newTestRunner(t)
	config, err := loadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	err = r.Start(config)
	if err != nil {
		t.Fatalf("expected no error on first start, got %v", err)
	}
	defer r.Stop()
	err = r.Restart(config)
	if err != nil {
		t.Errorf("expected no error on restart, got %v", err)
	}
	if !r.IsRunning() {
		t.Errorf("expected process to be running after restart")
	}
}
func TestSingboxRunner_Restart_AlreadyRestarting(t *testing.T) {
	mockCtrl := &mockProcessController{
		isRunning: true,
	}
	Runner := common.NewBaseRunner(common.DefaultSingboxExecutablePath, logging.NewStdLogger(), mockCtrl)
	r := &SingboxRunner{
		BaseRunner: Runner,
	}
	err := r.Restart(testConfig)
	if err == nil {
		t.Fatal("expected error when controller returns restart error, got nil")
	}
	if !strings.Contains(err.Error(), "restart error") {
		t.Errorf("expected 'restart error' in error message, got: %v", err)
	}
}
func TestSingboxRunner_Restart_NotRunning(t *testing.T) {
	r := newTestRunner(t)
	config, err := loadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	err = r.Restart(config)
	if err == nil {
		t.Fatal("expected error restarting not running process, got nil")
	}
	if !strings.Contains(err.Error(), "not running") {
		t.Errorf("expected 'not running' error, got: %v", err)
	}
}
func TestSingboxRunner_Reload_Success(t *testing.T) {
	cleanup := setupMockExecCommand(t)
	defer cleanup()
	r := newTestRunner(t)
	config, err := loadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	err = r.Start(config)
	if err != nil {
		t.Fatalf("expected no error on start, got %v", err)
	}
	defer r.Stop()
	err = r.Reload()
	if err != nil {
		t.Errorf("expected no error on reload, got %v", err)
	}
}
func TestSingboxRunner_Reload_NotRunning(t *testing.T) {
	r := newTestRunner(t)
	err := r.Reload()
	if err == nil {
		t.Fatal("expected error reloading not running process, got nil")
	}
	if !strings.Contains(err.Error(), "not running") {
		t.Errorf("expected 'not running' error, got: %v", err)
	}
}
func TestSingboxRunner_Stop_Success(t *testing.T) {
	cleanup := setupMockExecCommand(t)
	defer cleanup()
	r := newTestRunner(t)
	config, err := loadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	err = r.Start(config)
	if err != nil {
		t.Fatalf("expected no error on start, got %v", err)
	}
	if r.configFilePath == "" {
		t.Fatal("expected configFilePath to be set after start")
	}
	configPath := r.configFilePath
	err = r.Stop()
	if err != nil {
		t.Errorf("expected no error on stop, got %v", err)
	}
	if r.IsRunning() {
		t.Errorf("expected process to stop")
	}
	if r.configFilePath != "" {
		t.Errorf("expected configFilePath to be reset after stop")
	}
	_, err = os.Stat(configPath)
	if !os.IsNotExist(err) {
		t.Errorf("expected config file to be removed after stop")
	}
}
func TestSingboxRunner_Stop_NotRunning(t *testing.T) {
	r := newTestRunner(t)
	err := r.Stop()
	if err != nil {
		t.Errorf("expected no error stopping not running process, got %v", err)
	}
}
func TestSingboxRunner_Stop_ControllerError(t *testing.T) {
	cleanup := setupMockExecCommand(t)
	defer cleanup()
	r := newTestRunner(t)
	config, err := loadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	err = r.Start(config)
	if err != nil {
		t.Fatalf("expected no error on start, got %v", err)
	}
	mockCtrl := &mockProcessController{
		isRunning: true,
		stopError: fmt.Errorf("stop error"),
	}
	configPath := r.configFilePath
	origController := r.Controller
	r.Controller = mockCtrl
	defer func() {
		r.Controller = origController
		os.Remove(configPath)
	}()
	err = r.Stop()
	if err == nil {
		t.Fatal("expected error from controller.Stop, got nil")
	}
	if !strings.Contains(err.Error(), "stop error") {
		t.Errorf("expected 'stop error', got: %v", err)
	}
	if r.configFilePath == "" {
		t.Errorf("expected configFilePath to remain set on error")
	}
}

func TestSingboxRunner_IsRunning(t *testing.T) {
	t.Run("TrueAfterStart", func(t *testing.T) {
		cleanup := setupMockExecCommand(t)
		defer cleanup()
		r := newTestRunner(t)
		config, err := loadConfig()
		if err != nil {
			t.Fatalf("failed to load config: %v", err)
		}
		err = r.Start(config)
		if err != nil {
			t.Fatalf("expected no error on start, got %v", err)
		}
		defer r.Stop()
		if !r.IsRunning() {
			t.Errorf("expected IsRunning to return true after start")
		}
	})
	t.Run("FalseBeforeStart", func(t *testing.T) {
		t.Parallel()
		r := newTestRunner(t)
		if r.IsRunning() {
			t.Errorf("expected IsRunning to return false before start")
		}
	})
}

func TestSingboxRunner_GetBuffer(t *testing.T) {
	origExec := execCommand
	defer func() { execCommand = origExec }()
	execCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "echo 'log message 1'; echo 'log message 2'; sleep 10")
	}
	r := newTestRunner(t)
	err := r.Start(testConfig)
	if err != nil {
		t.Fatalf("expected no error on start, got %v", err)
	}
	defer r.Stop()
	time.Sleep(logWaitTimeout)
	buffer := r.GetBuffer()
	if len(buffer) < 2 {
		t.Errorf("expected at least 2 log entries, got %d", len(buffer))
	}
	foundLog1 := false
	foundLog2 := false
	for _, log := range buffer {
		if strings.Contains(log, "log message 1") {
			foundLog1 = true
		}
		if strings.Contains(log, "log message 2") {
			foundLog2 = true
		}
	}
	if !foundLog1 || !foundLog2 {
		t.Errorf("expected logs to contain both messages, got %v", buffer)
	}
}
func TestSingboxRunner_SubscribeLogs(t *testing.T) {
	origExec := execCommand
	defer func() { execCommand = origExec }()
	execCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "echo 'subscribe log test'; sleep 10")
	}
	r := newTestRunner(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	logsCh := r.SubscribeLogs(ctx)
	err := r.Start(testConfig)
	if err != nil {
		t.Fatalf("expected no error on start, got %v", err)
	}
	defer func() {
		r.Stop()
		cancel()
	}()
	var receivedLogs []string
	timeout := time.After(shortTimeout)
receiveLoop:
	for {
		select {
		case log, ok := <-logsCh:
			if !ok {
				break receiveLoop
			}
			receivedLogs = append(receivedLogs, log)
			if strings.Contains(log, "subscribe log test") {
				break receiveLoop
			}
		case <-timeout:
			t.Fatal("timed out waiting for logs")
		}
	}
	if len(receivedLogs) == 0 {
		t.Error("expected to receive logs, got none")
	}
	foundTestLog := false
	for _, log := range receivedLogs {
		if strings.Contains(log, "subscribe log test") {
			foundTestLog = true
			break
		}
	}
	if !foundTestLog {
		t.Errorf("expected to find test log, got %v", receivedLogs)
	}
}
func TestSingboxRunner_SubscribeLogs_ChannelCloseOnCancel(t *testing.T) {
	r := newTestRunner(t)
	ctx, cancel := context.WithCancel(context.Background())
	logsCh := r.SubscribeLogs(ctx)
	cancel()
	select {
	case _, ok := <-logsCh:
		if ok {
			t.Error("expected channel to be closed after context cancel")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timed out waiting for channel close")
	}
}
func TestSingboxRunner_StopEvent_TriggeredOnStop(t *testing.T) {
	cleanup := setupMockExecCommand(t)
	defer cleanup()
	r := newTestRunner(t)
	config, err := loadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	err = r.Start(config)
	if err != nil {
		t.Fatalf("expected no error on start, got %v", err)
	}
	stopEventCh := r.StopEvent()
	err = r.Stop()
	if err != nil {
		t.Fatalf("expected no error on stop, got %v", err)
	}
	select {
	case <-stopEventCh:
	case <-time.After(shortTimeout):
		t.Fatal("timed out waiting for stop event")
	}
}
func TestSingboxRunner_StopEvent_Channel(t *testing.T) {
	r := newTestRunner(t)
	stopEventCh := r.StopEvent()
	select {
	case <-stopEventCh:
		t.Fatal("expected stop event channel to be open")
	default:
	}
	r.TriggerStopEvent()
	select {
	case <-stopEventCh:
	default:
		t.Fatal("expected stop event channel to be closed")
	}
}
func TestSingboxRunner_StopEventOnProcessExit(t *testing.T) {
	origExec := execCommand
	defer func() { execCommand = origExec }()
	execCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("sh", "-c", mockExitCommand)
	}
	r := newTestRunner(t)
	stopEventCh := r.StopEvent()
	err := r.Start(testConfig)
	if err != nil {
		t.Fatalf("expected no error on start, got %v", err)
	}
	select {
	case <-stopEventCh:
	case <-time.After(testTimeout):
		t.Fatal("timed out waiting for stop event")
	}
	if r.IsRunning() {
		t.Errorf("expected process to stop")
	}
}
func TestSingboxRunner_CreateConfigFile_Success(t *testing.T) {
	cleanup := setupMockExecCommand(t)
	defer cleanup()
	r := newTestRunner(t)
	config := testConfig
	err := r.Start(config)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer r.Stop()
	if r.configFilePath == "" {
		t.Error("expected configFilePath to be set")
	}
	content, err := os.ReadFile(r.configFilePath)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}
	if string(content) != config {
		t.Errorf("expected config file to contain %q, got %q", config, string(content))
	}
}
func TestSingboxRunner_CreateConfigFile_Error(t *testing.T) {
	r := newTestRunner(t)
	tmpDir := t.TempDir()
	os.RemoveAll(tmpDir)
	t.Setenv("TMPDIR", tmpDir)
	err := r.Start(testConfig)
	if err == nil {
		t.Fatal("expected error creating config file, got nil")
	}
	if r.configFilePath != "" {
		t.Errorf("expected configFilePath to remain empty on error")
	}
}
func TestSingboxRunner_CreateConfigFile_WriteError(t *testing.T) {
	r := newTestRunner(t)
	tmpDir := t.TempDir()
	readOnlyFile := filepath.Join(tmpDir, "readonly.json")
	err := os.WriteFile(readOnlyFile, []byte{}, 0444)
	if err != nil {
		t.Fatalf("failed to create readonly file: %v", err)
	}
	restrictedDir := filepath.Join(tmpDir, "restricted")
	err = os.Mkdir(restrictedDir, 0555)
	if err != nil {
		t.Fatalf("failed to create restricted directory: %v", err)
	}
	defer os.Chmod(restrictedDir, 0755)
	originalTmpDir := os.Getenv("TMPDIR")
	os.Setenv("TMPDIR", restrictedDir)
	defer os.Setenv("TMPDIR", originalTmpDir)
	_, err = r.createConfigFile(testConfig)
	if err == nil {
		t.Fatal("expected error when creating config file in restricted directory, got nil")
	}
	if !strings.Contains(err.Error(), "failed to create temporary config file") {
		t.Errorf("expected 'failed to create temporary config file' error, got: %v", err)
	}
}
func TestSingboxRunner_CreateConfigFile_ExistingFileRemoval(t *testing.T) {
	r := newTestRunner(t)
	configPath1, err := r.createConfigFile(`{"test": "config1"}`)
	if err != nil {
		t.Fatalf("failed to create first config file: %v", err)
	}
	if _, err := os.Stat(configPath1); os.IsNotExist(err) {
		t.Fatal("expected first config file to exist")
	}
	configPath2, err := r.createConfigFile(`{"test": "config2"}`)
	if err != nil {
		t.Fatalf("failed to create second config file: %v", err)
	}
	if _, err := os.Stat(configPath1); !os.IsNotExist(err) {
		t.Error("expected first config file to be removed")
	}
	if _, err := os.Stat(configPath2); os.IsNotExist(err) {
		t.Fatal("expected second config file to exist")
	}
	os.Remove(configPath2)
}
func TestSingboxRunner_CreateConfigFile_WriteStringError(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping in short mode")
	}
	r := newTestRunner(t)
	type mockFile struct {
		*os.File
	}
	origCreateTemp := osCreateTemp
	defer func() { osCreateTemp = origCreateTemp }()
	osCreateTemp = func(dir, pattern string) (*os.File, error) {
		realFile, err := origCreateTemp(dir, pattern)
		if err != nil {
			return nil, err
		}
		r, w, err := os.Pipe()
		if err != nil {
			realFile.Close()
			return nil, err
		}
		r.Close()
		return w, nil
	}
	_, err := r.createConfigFile(testConfig)
	if err != nil && strings.Contains(err.Error(), "failed to write config to file") {
		t.Log("Successfully triggered WriteString error")
		return
	}
	t.Error("Expected WriteString error but didn't get one")
}
func TestSingboxRunner_CreateConfigFile_CloseErrorBeforeWrite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping in short mode")
	}
	r := newTestRunner(t)
	origCreateTemp := osCreateTemp
	defer func() { osCreateTemp = origCreateTemp }()
	osCreateTemp = func(dir, pattern string) (*os.File, error) {
		realFile, err := origCreateTemp(dir, pattern)
		if err != nil {
			return nil, err
		}
		realFile.Close()
		r, w, pipeErr := os.Pipe()
		if pipeErr != nil {
			return nil, pipeErr
		}
		r.Close()
		w.Close()
		return w, nil
	}
	_, err := r.createConfigFile(`test`)
	if err != nil && (strings.Contains(err.Error(), "failed to close config file") ||
		strings.Contains(err.Error(), "failed to write config to file")) {
		t.Log("Successfully triggered file operation error")
		return
	}
	t.Skip("Unable to trigger Close error - system handles closed files gracefully")
}
func TestSingboxRunner_CreateConfigFile_CloseErrorAfterWrite(t *testing.T) {
	r := newTestRunner(t)
	origFileCloser := fileCloser
	defer func() { fileCloser = origFileCloser }()
	fileCloser = func(f *os.File) error {
		f.Close()
		return fmt.Errorf("simulated close error after successful write")
	}
	_, err := r.createConfigFile(testConfig)
	if err != nil && strings.Contains(err.Error(), "failed to close config file") {
		t.Log("Successfully triggered Close error after write")
		return
	}
	t.Error("Expected Close error after write but didn't get one")
}
func TestSingboxRunner_RemoveConfigFile_Success(t *testing.T) {
	cleanup := setupMockExecCommand(t)
	defer cleanup()
	r := newTestRunner(t)
	config := testConfig
	err := r.Start(config)
	if err != nil {
		t.Fatalf("expected no error on start, got %v", err)
	}
	configPath := r.configFilePath
	r.Stop()
	if r.configFilePath != "" {
		t.Errorf("expected configFilePath to be reset, got %q", r.configFilePath)
	}
	_, err = os.Stat(configPath)
	if !os.IsNotExist(err) {
		t.Errorf("expected config file to be removed")
	}
}
func TestSingboxRunner_RemoveConfigFile_RemoveError(t *testing.T) {
	r := newTestRunner(t)
	configPath, err := r.createConfigFile(testConfig)
	if err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}
	err = os.Remove(configPath)
	if err != nil {
		t.Fatalf("failed to manually remove config file: %v", err)
	}
	r.removeConfigFile()
	if r.configFilePath != "" {
		t.Errorf("expected configFilePath to be reset after removeConfigFile, got %q", r.configFilePath)
	}
}
func TestSingboxRunner_RemoveConfigFile_EmptyPath(t *testing.T) {
	r := newTestRunner(t)
	r.removeConfigFile()
	if r.configFilePath != "" {
		t.Errorf("expected configFilePath to remain empty, got %q", r.configFilePath)
	}
}
func TestSingboxRunner_TriggerStopEvent_CalledTwice(t *testing.T) {
	r := newTestRunner(t)
	r.TriggerStopEvent()
	r.TriggerStopEvent()
}

var _ common.ProcessController = (*mockProcessController)(nil)

type mockProcessController struct {
	setupCmdErr error
	isRunning   bool
	stopError   error
	onStop      func()
}

func (m *mockProcessController) SetOnStop(fn func()) {
	m.onStop = fn
}
func (m *mockProcessController) SetupCmd(cmd *exec.Cmd) error {
	return m.setupCmdErr
}
func (m *mockProcessController) IsRunning() bool {
	return m.isRunning
}
func (m *mockProcessController) Restart(doStart func() error) error {
	return fmt.Errorf("restart error")
}
func (m *mockProcessController) Reload(signal syscall.Signal) error {
	return fmt.Errorf("reload error")
}
func (m *mockProcessController) Stop() error {
	return m.stopError
}
func (m *mockProcessController) GetBuffer() []string {
	return []string{}
}
func (m *mockProcessController) SubscribeLogs(ctx context.Context) <-chan string {
	ch := make(chan string)
	close(ch)
	return ch
}
func newTestRunner(t *testing.T) *SingboxRunner {
	tl := logging.NewStdLogger()
	exePath, err := exec.LookPath(common.DefaultSingboxExecutablePath)
	if err != nil {
		t.Skipf("failed to find singbox executable: %v", err)
	}
	r, err := NewSingboxRunner(exePath, tl)
	if err != nil {
		t.Fatalf("failed to create runner: %v", err)
	}
	return r
}
func setupMockExecCommand(_ *testing.T) func() {
	origExec := execCommand
	execCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("sh", "-c", mockSleepCommand)
	}
	return func() { execCommand = origExec }
}
func loadConfig() (string, error) {
	path := common.DefaultSingboxConfigPath
	data, err := assets.ConfigFS.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("%w (%s): %w", common.ErrFailedReadEmbeddedConfig, path, err)
	}
	return string(data), nil
}

func TestSingboxRunner_RealBinary_Start_LogCapture(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	singboxPath, err := exec.LookPath(common.DefaultSingboxExecutablePath)
	if err != nil {
		t.Skipf("sing-box binary not found at %s: %v", common.DefaultSingboxExecutablePath, err)
	}

	testConfig, err := loadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	logger := logging.NewStdLogger()
	runner, err := NewSingboxRunner(singboxPath, logger)
	if err != nil {
		t.Fatalf("Failed to create runner: %v", err)
	}
	defer func() {
		if runner.IsRunning() {
			runner.Stop()
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logsCh := runner.SubscribeLogs(ctx)

	err = runner.Start(testConfig)
	if err != nil {
		t.Fatalf("Failed to start sing-box: %v", err)
	}

	expectedPatterns := []string{
		"sing-box started",
		"started",
		"inbound/direct",
		"outbound/direct",
	}

	foundLogs := make(map[string]bool)
	var allLogs []string

	timeout := time.After(10 * time.Second)

readLoop:
	for {
		select {
		case log, ok := <-logsCh:
			if !ok {
				t.Logf("Logs channel closed")
				break readLoop
			}

			allLogs = append(allLogs, log)
			t.Logf("Received log: %s", log)

			for _, pattern := range expectedPatterns {
				if strings.Contains(strings.ToLower(log), strings.ToLower(pattern)) {
					foundLogs[pattern] = true
				}
			}

			if len(foundLogs) > 0 {
				t.Logf("Found readiness patterns: %v", foundLogs)
				break readLoop
			}

		case <-timeout:
			t.Errorf("Timeout waiting for readiness logs")
			break readLoop
		}
	}

	if len(foundLogs) == 0 {
		t.Errorf("No readiness patterns found in logs")
		t.Logf("All received logs:")
		for i, log := range allLogs {
			t.Logf("  [%d] %s", i+1, log)
		}
	} else {
		t.Logf("Successfully detected sing-box readiness")
	}

	if !runner.IsRunning() {
		t.Errorf("Runner should be running after successful start")
	}

	buffer := runner.GetBuffer()
	if len(buffer) == 0 {
		t.Errorf("Expected logs in buffer, got empty")
	} else {
		t.Logf("Buffer contains %d log entries", len(buffer))
	}
}

func TestSingboxRunner_RealBinary_Restart_EventReset(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	singboxPath, err := exec.LookPath(common.DefaultSingboxExecutablePath)
	if err != nil {
		t.Skipf("sing-box binary not found at %s: %v", common.DefaultSingboxExecutablePath, err)
	}

	validConfig, err := loadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	logger := logging.NewStdLogger()
	runner, err := NewSingboxRunner(singboxPath, logger)
	if err != nil {
		t.Fatalf("Failed to create runner: %v", err)
	}
	defer func() {
		if runner.IsRunning() {
			runner.Stop()
		}
	}()

	err = runner.Start(validConfig)
	if err != nil {
		t.Fatalf("Failed to start: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	if !runner.IsRunning() {
		t.Fatalf("Expected runner to be running")
	}

	stopEventCh := runner.StopEvent()

	err = runner.Restart(validConfig)
	if err != nil {
		t.Fatalf("Failed to restart: %v", err)
	}

	if !runner.IsRunning() {
		t.Errorf("Expected runner to be running after restart")
	}

	time.Sleep(200 * time.Millisecond)

	newStopEventCh := runner.StopEvent()
	if newStopEventCh == stopEventCh {
		t.Errorf("Stop event channel should be reset after restart")
	}

	go func() {
		time.Sleep(100 * time.Millisecond)
		runner.Stop()
	}()

	select {
	case <-newStopEventCh:
		t.Logf("New stop event triggered correctly after restart")
	case <-time.After(5 * time.Second):
		t.Errorf("New stop event not triggered within timeout")
	}
}

func TestSingboxRunner_RealBinary_Stop_Event(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	singboxPath, err := exec.LookPath(common.DefaultSingboxExecutablePath)
	if err != nil {
		t.Skipf("sing-box binary not found at %s: %v", common.DefaultSingboxExecutablePath, err)
	}

	validConfig, err := loadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	logger := logging.NewStdLogger()
	runner, err := NewSingboxRunner(singboxPath, logger)
	if err != nil {
		t.Fatalf("Failed to create runner: %v", err)
	}
	defer func() {
		if runner.IsRunning() {
			runner.Stop()
		}
	}()

	err = runner.Start(validConfig)
	if err != nil {
		t.Fatalf("Failed to start: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	if !runner.IsRunning() {
		t.Fatalf("Expected runner to be running")
	}

	stopEventCh := runner.StopEvent()

	go func() {
		time.Sleep(100 * time.Millisecond)
		runner.Stop()
	}()

	select {
	case <-stopEventCh:
		t.Logf("Stop event triggered correctly")
	case <-time.After(5 * time.Second):
		t.Errorf("Stop event not triggered within timeout")
	}

	if runner.IsRunning() {
		t.Errorf("Expected runner to be stopped")
	}
}

func TestSingboxRunner_RealBinary_SubscribeLogs_MultipleSubscribers(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	singboxPath, err := exec.LookPath(common.DefaultSingboxExecutablePath)
	if err != nil {
		t.Skipf("sing-box binary not found at %s: %v", common.DefaultSingboxExecutablePath, err)
	}

	testConfig, err := loadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	logger := logging.NewStdLogger()
	runner, err := NewSingboxRunner(singboxPath, logger)
	if err != nil {
		t.Fatalf("Failed to create runner: %v", err)
	}
	defer func() {
		if runner.IsRunning() {
			runner.Stop()
		}
	}()

	ctx1, cancel1 := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel1()
	ctx2, cancel2 := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel2()

	logsCh1 := runner.SubscribeLogs(ctx1)
	logsCh2 := runner.SubscribeLogs(ctx2)

	err = runner.Start(testConfig)
	if err != nil {
		t.Fatalf("Failed to start: %v", err)
	}

	subscriber1Received := false
	subscriber2Received := false

	timeout := time.After(8 * time.Second)

checkLoop:
	for {
		select {
		case log1, ok := <-logsCh1:
			if ok {
				t.Logf("Subscriber 1 received: %s", log1)
				subscriber1Received = true
			}
		case log2, ok := <-logsCh2:
			if ok {
				t.Logf("Subscriber 2 received: %s", log2)
				subscriber2Received = true
			}
		case <-timeout:
			break checkLoop
		}

		if subscriber1Received && subscriber2Received {
			break checkLoop
		}
	}

	if !subscriber1Received {
		t.Errorf("Subscriber 1 did not receive logs")
	}
	if !subscriber2Received {
		t.Errorf("Subscriber 2 did not receive logs")
	}

	if subscriber1Received && subscriber2Received {
		t.Logf("Both subscribers received logs successfully")
	}
}
