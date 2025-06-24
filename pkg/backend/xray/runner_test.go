package xray

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"slices"

	"github.com/highlight-apps/node-backend/assets"
	"github.com/highlight-apps/node-backend/backend/common"
	"github.com/highlight-apps/node-backend/logging"
)

func TestMain(m *testing.M) {
	realExec := exec.Command
	execCommand = func(name string, args ...string) *exec.Cmd {
		if filepath.Base(name) == "true" {
			return realExec(name, args...)
		}
		return realExec("sh", "-c", "echo '[Warning] core: Xray 1.2.3 started'; sleep 10")
	}
	os.Exit(m.Run())
}

func TestXrayRunner_New_Success(t *testing.T) {
	tl := logging.NewStdLogger()

	exePath, err := exec.LookPath(common.DefaultXrayExecutablePath)
	if err != nil {
		t.Fatalf("failed to find xray executable: %v", err)
	}

	assetsPath, err := filepath.Abs(filepath.Join("..", "..", common.DefaultAssetsPath))
	if err != nil {
		t.Fatalf("failed to resolve assets path: %v", err)
	}

	_, err = NewXrayRunner(exePath, assetsPath, tl)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestXrayRunner_New_DefaultLogger(t *testing.T) {
	exePath, err := exec.LookPath(common.DefaultXrayExecutablePath)
	if err != nil {
		t.Fatalf("failed to find xray executable: %v", err)
	}
	assetsPath, err := filepath.Abs(filepath.Join("..", "..", common.DefaultAssetsPath))
	if err != nil {
		t.Fatalf("failed to resolve assets path: %v", err)
	}
	b, err := NewXrayRunner(exePath, assetsPath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if b.Logger == nil {
		t.Errorf("expected default logger to be set")
	}
	if b.ExecutablePath != exePath {
		t.Errorf("expected executablePath to be %v, got %v", exePath, b.ExecutablePath)
	}
	if b.assetsPath != assetsPath {
		t.Errorf("expected assetsPath to be %v, got %v", assetsPath, b.assetsPath)
	}
}

func TestXrayRunner_New_CustomLogger(t *testing.T) {
	logger := logging.NewStdLogger()
	b := newTestRunnerWithLogger(t, logger)

	if b.Logger != logger {
		t.Errorf("expected logger to be set")
	}
}

func TestXrayRunner_New_ExecutablePathSet(t *testing.T) {
	b := newTestRunner(t)

	if b.ExecutablePath == "" {
		t.Errorf("expected executable path to be set")
	}
}

func TestXrayRunner_New_AssetsPathSet(t *testing.T) {
	b := newTestRunner(t)

	if b.assetsPath == "" {
		t.Errorf("expected assets path to be set")
	}
}

func TestXrayRunner_New_ControllerSet(t *testing.T) {
	exePath, err := exec.LookPath(common.DefaultXrayExecutablePath)
	if err != nil {
		t.Fatalf("failed to find xray executable: %v", err)
	}
	assetsPath, err := filepath.Abs(filepath.Join("..", "..", common.DefaultAssetsPath))
	if err != nil {
		t.Fatalf("failed to resolve assets path: %v", err)
	}
	b, err := NewXrayRunner(exePath, assetsPath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if b.Controller == nil {
		t.Errorf("expected controller to be set")
	}
	logger := logging.NewStdLogger()
	b2, err := NewXrayRunner(exePath, assetsPath, logger)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if b2.Controller == nil {
		t.Errorf("expected controller to be set for custom logger")
	}
}

func TestXrayRunner_Version_Success(t *testing.T) {
	b := newTestRunner(t)
	version, err := b.Version()

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if version != "25.5.16" {
		t.Errorf("expected version '25.5.16', got %q", version)
	}
}

func TestXrayRunner_Version_Error(t *testing.T) {
	orig := runCombinedOutput
	defer func() { runCombinedOutput = orig }()
	runCombinedOutput = func(name string, args ...string) ([]byte, error) {
		return nil, fmt.Errorf("exec error")
	}
	b := newTestRunner(t)
	_, err := b.Version()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, common.ErrFailedToGetVersion) {
		t.Errorf("expected ErrFailedToGetVersion, got %v", err)
	}
}

func TestXrayRunner_Start_Success(t *testing.T) {
	b := newTestRunner(t)

	config, err := loadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	err = b.Start(config)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestXrayRunner_Start_AlreadyRunning(t *testing.T) {
	b := newTestRunner(t)

	config, err := loadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	err = b.Start(config)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	err = b.Start(config)
	if err != common.ErrProcessAlreadyRunning {
		t.Errorf("expected ErrRunnerAlreadyRunning, got %v", err)
	}
}

func TestXrayRunner_Start_InvalidExecutablePath(t *testing.T) {
	b := newTestRunnerWithWrongPath()

	config, err := loadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	err = b.Start(config)
	if err == common.ErrFailedToStartRunner {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestXrayRunner_Start_CLIArgsAndEnv(t *testing.T) {
	saved := execCommand
	defer func() { execCommand = saved }()
	var gotArgs []string
	var stubCmd *exec.Cmd
	execCommand = func(name string, args ...string) *exec.Cmd {
		gotArgs = append([]string{name}, args...)
		stubCmd = exec.Command("bash", "-c", fmt.Sprintf("echo '%s'", "[Warning] core: Xray 1.2.3 started"))
		return stubCmd
	}

	fakeAssets := "/my/assets"
	tl := logging.NewStdLogger()
	ctrl := common.NewProcessController(tl)
	Runner := common.NewBaseRunner("xraybin", tl, ctrl)
	b := &XrayRunner{
		BaseRunner: Runner,
		assetsPath: fakeAssets,
	}

	config, err := loadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if err := b.Start(config); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := []string{"xraybin", "run", "-config", "stdin:", "--location-asset", fakeAssets}
	if !reflect.DeepEqual(gotArgs, expected) {
		t.Errorf("expected args %v, got %v", expected, gotArgs)
	}

	found := slices.Contains(stubCmd.Env, "XRAY_LOCATION_ASSET="+fakeAssets)
	if !found {
		t.Errorf("expected env to contain XRAY_LOCATION_ASSET=%s, got %v", fakeAssets, stubCmd.Env)
	}
}

func TestXrayRunner_Start_ErrorOnStart(t *testing.T) {
	saved := execCommand
	defer func() { execCommand = saved }()
	execCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("does-not-exist")
	}

	tl := logging.NewStdLogger()
	ctrl := common.NewProcessController(tl)
	Runner := common.NewBaseRunner("xraybin", tl, ctrl)
	b := &XrayRunner{
		BaseRunner: Runner,
		assetsPath: "",
	}

	config, err := loadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	err = b.Start(config)
	if err == nil {
		t.Fatal("expected ErrFailedToStartRunner, got nil")
	}
	if !errors.Is(err, common.ErrFailedToStartRunner) {
		t.Errorf("expected ErrFailedToStartRunner, got %v", err)
	}
}

func TestXrayRunner_Start_ErrorOnStdinPipe(t *testing.T) {
	saved := execCommand
	defer func() { execCommand = saved }()
	execCommand = func(name string, args ...string) *exec.Cmd {
		cmd := exec.Command("true")
		cmd.Stdin = os.Stdin
		return cmd
	}
	tl := logging.NewStdLogger()
	ctrl := common.NewProcessController(logging.NewStdLogger())
	Runner := common.NewBaseRunner("xraybin", tl, ctrl)
	b := &XrayRunner{
		BaseRunner: Runner,
		assetsPath: "",
	}

	config, err := loadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	err = b.Start(config)
	if err == nil {
		t.Fatal("expected ErrFailedToStartRunner, got nil")
	}
	if !errors.Is(err, common.ErrFailedToStartRunner) {
		t.Errorf("expected ErrFailedToStartRunner, got %v", err)
	}
}

func TestXrayRunner_Start_ErrorOnStdoutPipe(t *testing.T) {
	saved := execCommand
	defer func() { execCommand = saved }()
	execCommand = func(name string, args ...string) *exec.Cmd {
		cmd := exec.Command("true")
		cmd.Stdout = os.Stdout
		return cmd
	}
	tl := logging.NewStdLogger()
	ctrl := common.NewProcessController(tl)
	Runner := common.NewBaseRunner("xraybin", tl, ctrl)
	b := &XrayRunner{
		BaseRunner: Runner,
		assetsPath: "",
	}

	config, err := loadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	err = b.Start(config)
	if err == nil {
		t.Fatal("expected ErrFailedToStartRunner, got nil")
	}
	if !errors.Is(err, common.ErrFailedToStartRunner) {
		t.Errorf("expected ErrFailedToStartRunner, got %v", err)
	}
}

func TestXrayRunner_Start_ErrorOnStderrPipe(t *testing.T) {
	saved := execCommand
	defer func() { execCommand = saved }()
	execCommand = func(name string, args ...string) *exec.Cmd {
		cmd := exec.Command("true")
		cmd.Stderr = os.Stderr
		return cmd
	}
	tl := logging.NewStdLogger()
	ctrl := common.NewProcessController(tl)
	Runner := common.NewBaseRunner("xraybin", tl, ctrl)
	b := &XrayRunner{
		BaseRunner: Runner,
		assetsPath: "",
	}

	config, err := loadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	err = b.Start(config)
	if err == nil {
		t.Fatal("expected ErrFailedToStartRunner, got nil")
	}
	if !errors.Is(err, common.ErrFailedToStartRunner) {
		t.Errorf("expected ErrFailedToStartRunner, got %v", err)
	}
}

func TestXrayRunner_Start_WriteConfigError(t *testing.T) {
	savedPipe := newStdinPipe
	defer func() { newStdinPipe = savedPipe }()
	expectedErr := fmt.Errorf("write fail")
	newStdinPipe = func(cmd *exec.Cmd) (io.WriteCloser, error) {
		return errWriterCloser{err: expectedErr}, nil
	}
	savedExec := execCommand
	defer func() { execCommand = savedExec }()
	execCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "echo '[Warning] core: Xray 1.2.3 started'; sleep 10")
	}

	tl := logging.NewStdLogger()
	ctrl := common.NewProcessController(tl)
	Runner := common.NewBaseRunner("xraybin", tl, ctrl)
	b := &XrayRunner{
		BaseRunner: Runner,
		assetsPath: "",
	}

	config, err := loadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if err := b.Start(config); err != nil {
		t.Fatalf("expected no error from Start, got %v", err)
	}
	time.Sleep(20 * time.Millisecond)
}

func TestXrayRunner_Start_TimeoutWaitingStartupLog(t *testing.T) {
	tl := logging.NewStdLogger()
	exePath, err := exec.LookPath("true")
	if err != nil {
		t.Skipf("true executable not found: %v", err)
	}
	config, err := loadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	origExecCommand := execCommand
	defer func() { execCommand = origExecCommand }()

	execCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("true")
	}

	ctrl := common.NewProcessController(tl)
	Runner := common.NewBaseRunner(exePath, tl, ctrl)
	b := &XrayRunner{
		BaseRunner: Runner,
		assetsPath: "",
	}

	err = b.Start(config)
	if err == nil {
		t.Fatalf("expected timeout error, got nil")
	}
	if !errors.Is(err, common.ErrFailedToStartRunner) {
		t.Errorf("expected ErrFailedToStartRunner, got %v", err)
	}

	expectedMsgs := []string{"timeout", "startup log not found", "process stopped unexpectedly"}
	msgFound := false
	for _, msg := range expectedMsgs {
		if strings.Contains(err.Error(), msg) {
			msgFound = true
			break
		}
	}
	if !msgFound {
		t.Errorf("expected timeout/process error message, got: %v", err)
	}
}

func TestXrayRunner_Start_ActualTimeoutWaitingStartupLog(t *testing.T) {
	tl := logging.NewStdLogger()

	origExecCommand := execCommand
	origTimeout := startupTimeout
	defer func() {
		execCommand = origExecCommand
		startupTimeout = origTimeout
	}()

	startupTimeout = 100 * time.Millisecond

	execCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "echo 'Some other message'; sleep 10")
	}

	ctrl := common.NewProcessController(tl)
	Runner := common.NewBaseRunner("xray", tl, ctrl)
	b := &XrayRunner{
		BaseRunner: Runner,
		assetsPath: "assets",
	}

	config, err := loadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	start := time.Now()
	err = b.Start(config)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}

	if !errors.Is(err, common.ErrFailedToStartRunner) {
		t.Errorf("expected ErrFailedToStartRunner, got %v", err)
	}

	if !strings.Contains(err.Error(), "startup log not found within timeout") {
		t.Errorf("expected 'startup log not found within timeout' error, got: %v", err)
	}

	if elapsed < startupTimeout || elapsed > startupTimeout*2 {
		t.Errorf("expected elapsed time to be around %v, got %v", startupTimeout, elapsed)
	}

	b.Stop()
}

func TestXrayRunner_Restart_Success(t *testing.T) {
	b := newTestRunner(t)

	config, err := loadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	err = b.Start(config)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	err = b.Restart(config)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestXrayRunner_Restart_NotRunning(t *testing.T) {
	b := newTestRunner(t)

	config, err := loadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	err = b.Restart(config)
	if err != common.ErrProcessNotRunning {
		t.Errorf("expected ErrProcessNotRunning, got %v", err)
	}
}

func TestXrayRunner_Reload_Success(t *testing.T) {
	b := newTestRunner(t)
	if err := b.Reload(); err != nil {
		t.Errorf("expected no error from Reload, got %v", err)
	}
}

func TestXrayRunner_Stop_Success(t *testing.T) {
	b := newTestRunner(t)

	config, err := loadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if err := b.Start(config); err != nil {
		t.Fatalf("expected no error on start, got %v", err)
	}
	if err := b.Stop(); err != nil {
		t.Errorf("expected no error on stop, got %v", err)
	}
}

func TestXrayRunner_Stop_NotRunning(t *testing.T) {
	b := newTestRunner(t)
	err := b.Stop()
	if !errors.Is(err, common.ErrProcessNotRunning) {
		t.Errorf("expected ErrProcessNotRunning, got %v", err)
	}
}

func TestXrayRunner_IsRunning(t *testing.T) {
	b := newTestRunner(t)

	config, err := loadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	err = b.Start(config)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !b.IsRunning() {
		t.Errorf("expected IsRunning to return true")
	}
}

func TestXrayRunner_GetBuffer(t *testing.T) {
	b := newTestRunner(t)

	config, err := loadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if err := b.Start(config); err != nil {
		t.Fatalf("expected no error on start, got %v", err)
	}
	buf := b.GetBuffer()
	found := false
	for _, line := range buf {
		if strings.Contains(line, "[Warning] core: Xray") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected startup log in buffer, got %v", buf)
	}
}

func TestXrayRunner_SubscribeLogs(t *testing.T) {
	b := newTestRunner(t)
	ctx, cancel := context.WithCancel(context.Background())
	ch := b.SubscribeLogs(ctx)
	defer cancel()

	config, err := loadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if err := b.Start(config); err != nil {
		t.Fatalf("expected no error on start, got %v", err)
	}
	timeout := time.After(4 * time.Second)
	for {
		select {
		case line, ok := <-ch:
			if !ok {
				t.Fatalf("channel closed before receiving logs")
			}
			if strings.Contains(line, "[Warning] core: Xray") {
				return
			}
		case <-timeout:
			t.Fatal("timeout waiting for startup log on subscribe channel")
		}
	}
}

func TestXrayRunner_UnsubscribeClosesChannel(t *testing.T) {
	b := newTestRunner(t)
	ctx, cancel := context.WithCancel(context.Background())
	ch := b.SubscribeLogs(ctx)
	cancel()
	_, ok := <-ch
	if ok {
		t.Error("expected channel to be closed after unsubscribe")
	}
}

func TestXrayRunner_StopEvent_TriggeredOnStop(t *testing.T) {
	b := newTestRunner(t)

	config, err := loadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	err = b.Start(config)
	if err != nil {
		t.Fatalf("expected no error on start, got %v", err)
	}

	stopEventCh := b.StopEvent()

	go func() {
		time.Sleep(100 * time.Millisecond)
		b.Stop()
	}()

	select {
	case <-stopEventCh:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for stop event")
	}
}

func TestXrayRunner_StopEvent_Channel(t *testing.T) {
	b := newTestRunner(t)
	ch := b.StopEvent()
	if ch == nil {
		t.Error("expected non-nil stop event channel")
	}

	select {
	case <-ch:
		t.Error("stop event channel should not be ready before stop")
	default:
	}
}

func TestXrayRunner_StopEventOnProcessExit(t *testing.T) {
	origExec := execCommand
	defer func() { execCommand = origExec }()
	execCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "echo '[Warning] core: Xray 1.2.3 started'; exit 0")
	}

	b := newTestRunner(t)
	stopEventCh := b.StopEvent()

	config, err := loadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	err = b.Start(config)
	if err != nil {
		t.Fatalf("expected no error on start, got %v", err)
	}

	select {
	case <-stopEventCh:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for stop event after process exit")
	}

	if b.IsRunning() {
		t.Error("expected IsRunning to return false after process exit")
	}
}

type errWriterCloser struct{ err error }

func (e errWriterCloser) Write(p []byte) (int, error) { return 0, e.err }
func (e errWriterCloser) Close() error                { return nil }

func newTestRunner(t *testing.T) *XrayRunner {
	tl := logging.NewStdLogger()
	exePath, err := exec.LookPath(common.DefaultXrayExecutablePath)
	if err != nil {
		t.Fatalf("failed to find xray executable: %v", err)
	}

	assetsPath, err := filepath.Abs(filepath.Join("..", "..", common.DefaultAssetsPath))
	if err != nil {
		t.Fatalf("failed to resolve assets path: %v", err)
	}
	ctrl := common.NewProcessController(tl)
	Runner := common.NewBaseRunner(exePath, tl, ctrl)
	b := &XrayRunner{
		BaseRunner: Runner,
		assetsPath: assetsPath,
	}
	return b
}

func newTestRunnerWithLogger(t *testing.T, logger logging.Logger) *XrayRunner {
	exePath, err := exec.LookPath(common.DefaultXrayExecutablePath)
	if err != nil {
		t.Fatalf("failed to find xray executable: %v", err)
	}
	ctrl := common.NewProcessController(logger)
	Runner := common.NewBaseRunner(exePath, logger, ctrl)
	b := &XrayRunner{
		BaseRunner: Runner,
		assetsPath: "",
	}
	return b
}

func newTestRunnerWithWrongPath() *XrayRunner {
	tl := logging.NewStdLogger()
	exePath, _ := exec.LookPath("xray-wrong")
	ctrl := common.NewProcessController(tl)
	Runner := common.NewBaseRunner(exePath, tl, ctrl)
	b := &XrayRunner{
		BaseRunner: Runner,
		assetsPath: "",
	}
	return b
}

func loadConfig() (string, error) {
	var path string
	path = common.DefaultXrayConfigPath

	data, err := assets.ConfigFS.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("%w (%s): %w", common.ErrFailedReadEmbeddedConfig, path, err)
	}
	return string(data), nil
}
