//go:build !windows

package runner

import (
	"os/exec"
	"syscall"
)

func setupSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func sendTermSignal(pid int) {
	syscall.Kill(-pid, syscall.SIGTERM)
}

func sendKillSignal(pid int) {
	syscall.Kill(-pid, syscall.SIGKILL)
}
