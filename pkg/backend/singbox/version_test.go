package singbox

import (
	"errors"
	"fmt"
	"os/exec"
	"testing"

	"github.com/highlight-apps/node-backend/backend/common"
	"github.com/highlight-apps/node-backend/logging"
)

func TestGetSingboxVersion_Success(t *testing.T) {
	orig := runCombinedOutput
	defer func() { runCombinedOutput = orig }()
	runCombinedOutput = func(name string, args ...string) ([]byte, error) {
		return []byte("sing-box version 1.11.10"), nil
	}

	version, err := getSingboxVersion("fake-path")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if version != "1.11.10" {
		t.Errorf("expected version '1.11.10', got %q", version)
	}
}

func TestGetSingboxVersion_ExecError(t *testing.T) {
	orig := runCombinedOutput
	defer func() { runCombinedOutput = orig }()
	runCombinedOutput = func(name string, args ...string) ([]byte, error) {
		return nil, fmt.Errorf("exec error")
	}

	_, err := getSingboxVersion("fake-path")
	if err == nil {
		t.Fatal("expected exec error, got nil")
	}
}

func TestGetSingboxVersion_UnmatchedOutput(t *testing.T) {
	orig := runCombinedOutput
	defer func() { runCombinedOutput = orig }()
	runCombinedOutput = func(name string, args ...string) ([]byte, error) {
		return []byte("invalid output"), nil
	}

	version, err := getSingboxVersion("fake-path")
	if err == nil {
		t.Fatalf("expected error for unmatched output, got nil")
	}
	if version != "" {
		t.Errorf("expected empty version on error, got %q", version)
	}
}

func TestSingboxRunner_Version(t *testing.T) {
	tests := []struct {
		name          string
		mockOutput    []byte
		mockError     error
		expectedVer   string
		expectedErrIs error
		shouldError   bool
	}{
		{
			name:        "Success",
			mockOutput:  []byte("sing-box version 1.11.10"),
			mockError:   nil,
			expectedVer: "1.11.10",
			shouldError: false,
		},
		{
			name:          "Error",
			mockOutput:    nil,
			mockError:     fmt.Errorf("exec error"),
			expectedErrIs: common.ErrFailedToGetVersion,
			shouldError:   true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			orig := runCombinedOutput
			defer func() { runCombinedOutput = orig }()
			runCombinedOutput = func(name string, args ...string) ([]byte, error) {
				return tt.mockOutput, tt.mockError
			}

			r := newVersionTestRunner(t)
			version, err := r.Version()

			if tt.shouldError {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.expectedErrIs != nil && !errors.Is(err, tt.expectedErrIs) {
					t.Errorf("expected error %v, got %v", tt.expectedErrIs, err)
				}
			} else {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if version != tt.expectedVer {
					t.Errorf("expected version %q, got %q", tt.expectedVer, version)
				}
			}
		})
	}
}

func newVersionTestRunner(t *testing.T) *SingboxRunner {
	exePath, err := exec.LookPath(common.DefaultSingboxExecutablePath)
	if err != nil {
		t.Fatalf("failed to find singbox executable: %v", err)
	}

	logger := logging.NewStdLogger()
	pc := common.NewProcessController(logger)
	baseRunner := common.NewBaseRunner(exePath, logger, pc)

	return &SingboxRunner{
		BaseRunner: baseRunner,
	}
}
