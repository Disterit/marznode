package hysteria2

import (
	"fmt"
	"os/exec"
)

func getVersion(hysteriaPath string) (string, error) {
	cmd := exec.Command(hysteriaPath, "version")

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("error to get version: %v\n", err)
		return "", err
	}

	return string(output), err
}
