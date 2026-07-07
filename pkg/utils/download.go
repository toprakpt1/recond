package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func DownloadFile(url, dest string) error {
	dir := filepath.Dir(dest)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	if _, err := exec.LookPath("curl"); err == nil {
		cmd := exec.Command("curl", "-fsSL", "-o", dest, url)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err == nil {
			return nil
		}
	}

	if _, err := exec.LookPath("wget"); err == nil {
		cmd := exec.Command("wget", "-q", "-O", dest, url)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err == nil {
			return nil
		}
	}

	return fmt.Errorf("neither curl nor wget is available")
}
