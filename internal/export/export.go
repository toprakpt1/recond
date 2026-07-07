package export

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/recond/internal/storage"
)

type Exporter struct {
	store *storage.Storage
}

func NewExporter(store *storage.Storage) *Exporter {
	return &Exporter{store: store}
}

type ExportResult struct {
	Items []string `json:"items"`
	Count int      `json:"count"`
	Type  string   `json:"type"`
}

func (e *Exporter) ExportJobResults(jobID, outputType, format, outputPath string) error {
	dataDir := e.getDataDir(jobID)

	if outputType == "all" {
		types := []string{"subdomains", "alive", "urls", "directories"}
		for _, t := range types {
			if err := e.exportType(jobID, dataDir, t, format, ""); err != nil {
				continue
			}
		}
		return nil
	}

	return e.exportType(jobID, dataDir, outputType, format, outputPath)
}

func (e *Exporter) exportType(jobID, dataDir, outputType, format, outputPath string) error {
	items, err := e.readOutputFile(dataDir, outputType)
	if err != nil {
		return fmt.Errorf("no %s output found for job %s", outputType, jobID)
	}

	result := &ExportResult{
		Items: items,
		Count: len(items),
		Type:  outputType,
	}

	var output string
	switch format {
	case "json":
		data, _ := json.MarshalIndent(result, "", "  ")
		output = string(data)
	case "csv":
		output = e.toCSV(items, outputType)
	default:
		output = strings.Join(items, "\n")
	}

	if outputPath != "" {
		dir := filepath.Dir(outputPath)
		if dir != "" {
			os.MkdirAll(dir, 0755)
		}
		return os.WriteFile(outputPath, []byte(output), 0644)
	}

	fmt.Print(output)
	return nil
}

func (e *Exporter) readOutputFile(dataDir, outputType string) ([]string, error) {
	var fileName string
	switch outputType {
	case "subdomains":
		fileName = "subfinder.txt"
	case "alive":
		fileName = "httpx.txt"
	case "urls":
		fileName = "katana.txt"
	case "directories":
		fileName = "directories.json"
	default:
		return nil, fmt.Errorf("unknown output type: %s", outputType)
	}

	filePath := filepath.Join(dataDir, fileName)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var lines []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines, nil
}

func (e *Exporter) toCSV(items []string, outputType string) string {
	var sb strings.Builder
	w := csv.NewWriter(&sb)

	w.Write([]string{"type", "value", "status"})

	for _, item := range items {
		w.Write([]string{outputType, item, "found"})
	}

	w.Flush()
	return sb.String()
}

func (e *Exporter) getDataDir(jobID string) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".recond", "jobs", jobID)
}

func (e *Exporter) ListOutputs(jobID string) ([]string, error) {
	dataDir := e.getDataDir(jobID)

	var outputs []string
	types := map[string]string{
		"subdomains":  "subfinder.txt",
		"alive":       "httpx.txt",
		"crawled":     "katana.txt",
		"urls":        "gau.txt",
		"directories": "directories.json",
	}

	for name, file := range types {
		path := filepath.Join(dataDir, file)
		if _, err := os.Stat(path); err == nil {
			data, _ := os.ReadFile(path)
			lines := strings.Split(string(data), "\n")
			count := 0
			for _, l := range lines {
				if strings.TrimSpace(l) != "" {
					count++
				}
			}
			outputs = append(outputs, fmt.Sprintf("%s: %d items (%s)", name, count, path))
		}
	}

	return outputs, nil
}
