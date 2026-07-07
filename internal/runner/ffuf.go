package runner

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/toprakpt1/recond/pkg/utils"
)

type FfufRunner struct{}

func (r *FfufRunner) Name() string {
	return "ffuf"
}

func (r *FfufRunner) IsInstalled() bool {
	_, err := exec.LookPath("ffuf")
	return err == nil
}

func (r *FfufRunner) BuildCommand(opts RunOptions) ([]string, error) {
	args := []string{"ffuf"}

	if opts.Target == "" {
		return nil, fmt.Errorf("target URL is required")
	}

	args = append(args, "-u", opts.Target+"/FUZZ")

	wordlist := opts.Wordlist
	if wordlist == "" {
		wordlist = "~/.recond/wordlists/common.txt"
	}
	wordlist = expandPath(wordlist)

	if _, err := os.Stat(wordlist); os.IsNotExist(err) {
		log.Printf("[ffuf] wordlist not found, downloading: %s", wordlist)
		url := "https://raw.githubusercontent.com/danielmiessler/SecLists/master/Discovery/Web-Content/common.txt"
		if err := utils.DownloadFile(url, wordlist); err != nil {
			return nil, fmt.Errorf("wordlist not found and download failed: %w", err)
		}
		log.Printf("[ffuf] wordlist downloaded: %s", wordlist)
	}
	args = append(args, "-w", wordlist)

	if opts.OutputFile == "" {
		opts.OutputFile = filepath.Join(opts.OutputDir, "directories.json")
	}
	args = append(args, "-o", opts.OutputFile, "-of", "json")

	if opts.Concurrency > 0 {
		args = append(args, "-t", strconv.Itoa(opts.Concurrency))
	}

	if opts.Timeout > 0 {
		args = append(args, "-p", fmt.Sprintf("%f", float64(opts.Timeout.Seconds())/float64(opts.Concurrency)))
	}

	args = append(args, "-s", "-mc", "200,301,302,403")

	return args, nil
}

func (r *FfufRunner) ParseOutput(data []byte) ([]string, error) {
	var results []string
	seen := make(map[string]bool)

	var ffufOutput struct {
		Results []struct {
			URL       string `json:"url"`
			Status    int    `json:"status"`
			Length    int    `json:"length"`
			Words     int    `json:"words"`
			Recursive bool   `json:"recursive"`
		} `json:"results"`
	}

	if err := json.Unmarshal(data, &ffufOutput); err == nil {
		for _, result := range ffufOutput.Results {
			key := fmt.Sprintf("%s:%d", result.URL, result.Status)
			if !seen[key] {
				seen[key] = true
				results = append(results, fmt.Sprintf("%s [%d] (%d words, %d bytes)", result.URL, result.Status, result.Words, result.Length))
			}
		}
		return results, nil
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.Contains(line, "Status:") || strings.Contains(line, "[200]") {
			if !seen[line] {
				seen[line] = true
				results = append(results, line)
			}
		}
	}

	return results, nil
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}
