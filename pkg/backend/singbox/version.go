package singbox

import (
	"os/exec"
	"regexp"

	"github.com/highlight-apps/node-backend/backend/common"
)

var singboxVersionRegex = regexp.MustCompile(`^sing-box version (\d+\.\d+\.\d+)`)

var runCombinedOutput = func(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).CombinedOutput()
}

func getSingboxVersion(executablePath string) (string, error) {
	output, err := runCombinedOutput(executablePath, "version")
	if err != nil {
		return "", err
	}
	matches := singboxVersionRegex.FindStringSubmatch(string(output))
	if len(matches) < 2 {
		return "", common.ErrFailedToParseVersion
	}
	return matches[1], nil
}
