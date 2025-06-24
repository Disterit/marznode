package xray

import (
	"os/exec"
	"regexp"

	"github.com/highlight-apps/node-backend/backend/common"
)

var runCombinedOutput = func(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).CombinedOutput()
}

var xrayVersionRegex = regexp.MustCompile(`^Xray (\d+\.\d+\.\d+)`)

func getXrayVersion(executablePath string) (string, error) {
	output, err := runCombinedOutput(executablePath, "version")
	if err != nil {
		return "", err
	}

	outputString := string(output)

	matches := xrayVersionRegex.FindStringSubmatch(outputString)
	if len(matches) < 2 {
		return "", common.ErrFailedToParseVersion
	}
	return matches[1], nil
}
