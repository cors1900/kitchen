package exec

import (
	"os/exec"
)

func Run(name string, args ...string) (exitCode int, output []byte, err error) {
	cmd := exec.Command(name, args...)
	output, err = cmd.CombinedOutput()

	if err == nil {
		return 0, output, nil
	}
	// 解析退出码
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode(), output, nil
	}
	return -1, output, err
}
