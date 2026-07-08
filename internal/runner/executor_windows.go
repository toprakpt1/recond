//go:build windows

package runner

import (
	"os"
	"os/exec"
)

func setupSysProcAttr(cmd *exec.Cmd) {
	// Windows does not support Setpgid; no-op.
}

func sendTermSignal(pid int) {
	// Windows has no SIGTERM equivalent. Use Process.Kill() which calls
	// TerminateProcess for an immediate shutdown.
	if p, err := os.FindProcess(pid); err == nil {
		p.Kill()
	}
}

func sendKillSignal(pid int) {
	if p, err := os.FindProcess(pid); err == nil {
		p.Kill()
	}
}
