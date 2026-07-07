package template

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

type Template struct {
	Name        string    `yaml:"name"`
	Description string    `yaml:"description"`
	Version     string    `yaml:"version,omitempty"`
	Author      string    `yaml:"author,omitempty"`
	Steps       []StepDef `yaml:"steps"`
	CreatedAt   time.Time `yaml:"created_at,omitempty"`
	UpdatedAt   time.Time `yaml:"updated_at,omitempty"`
}

type StepDef struct {
	Name     string   `yaml:"name"`
	Tool     string   `yaml:"tool"`
	Order    int      `yaml:"order"`
	Input    string   `yaml:"input,omitempty"`
	Output   string   `yaml:"output,omitempty"`
	DependsOn []string `yaml:"depends_on,omitempty"`
	Parallel bool     `yaml:"parallel,omitempty"`
}

var DefaultTemplates = map[string]*Template{
	"full-recon": {
		Name:        "full-recon",
		Description: "Full reconnaissance pipeline with all tools",
		Steps: []StepDef{
			{Name: "subdomain-discovery", Tool: "subfinder", Order: 1},
			{Name: "alive-check", Tool: "httpx", Order: 2, DependsOn: []string{"subdomain-discovery"}},
			{Name: "crawling", Tool: "katana", Order: 3, DependsOn: []string{"alive-check"}, Parallel: true},
			{Name: "url-collection", Tool: "gau", Order: 3, DependsOn: []string{"alive-check"}, Parallel: true},
			{Name: "directory-fuzzing", Tool: "ffuf", Order: 4, DependsOn: []string{"alive-check"}},
		},
	},
	"subdomain-only": {
		Name:        "subdomain-only",
		Description: "Subdomain discovery only",
		Steps: []StepDef{
			{Name: "subdomain-discovery", Tool: "subfinder", Order: 1},
		},
	},
	"alive-check": {
		Name:        "alive-check",
		Description: "Subdomain discovery + alive check",
		Steps: []StepDef{
			{Name: "subdomain-discovery", Tool: "subfinder", Order: 1},
			{Name: "alive-check", Tool: "httpx", Order: 2, DependsOn: []string{"subdomain-discovery"}},
		},
	},
	"url-collection": {
		Name:        "url-collection",
		Description: "Full URL collection pipeline",
		Steps: []StepDef{
			{Name: "subdomain-discovery", Tool: "subfinder", Order: 1},
			{Name: "alive-check", Tool: "httpx", Order: 2, DependsOn: []string{"subdomain-discovery"}},
			{Name: "crawling", Tool: "katana", Order: 3, DependsOn: []string{"alive-check"}},
			{Name: "url-collection", Tool: "gau", Order: 4, DependsOn: []string{"alive-check"}},
		},
	},
	"directory-fuzz": {
		Name:        "directory-fuzz",
		Description: "Subdomain + alive + directory fuzzing",
		Steps: []StepDef{
			{Name: "subdomain-discovery", Tool: "subfinder", Order: 1},
			{Name: "alive-check", Tool: "httpx", Order: 2, DependsOn: []string{"subdomain-discovery"}},
			{Name: "directory-fuzzing", Tool: "ffuf", Order: 3, DependsOn: []string{"alive-check"}},
		},
	},
}

func TemplatesDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".recond", "templates")
}

func LoadTemplate(name string) (*Template, error) {
	if t, ok := DefaultTemplates[name]; ok {
		return t, nil
	}

	dir := TemplatesDir()
	path := filepath.Join(dir, name+".yaml")

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("template '%s' not found", name)
	}

	var t Template
	if err := yaml.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	return &t, nil
}

func ListTemplates() ([]*Template, error) {
	var templates []*Template

	for _, t := range DefaultTemplates {
		templates = append(templates, t)
	}

	dir := TemplatesDir()
	entries, err := os.ReadDir(dir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
				continue
			}

			data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
			if err != nil {
				continue
			}

			var t Template
			if err := yaml.Unmarshal(data, &t); err != nil {
				continue
			}

			templates = append(templates, &t)
		}
	}

	return templates, nil
}

func CreateTemplate(name string, t *Template) error {
	dir := TemplatesDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	t.Name = name
	t.CreatedAt = time.Now()
	t.UpdatedAt = time.Now()

	data, err := yaml.Marshal(t)
	if err != nil {
		return err
	}

	path := filepath.Join(dir, name+".yaml")
	return os.WriteFile(path, data, 0644)
}

func DeleteTemplate(name string) error {
	if _, ok := DefaultTemplates[name]; ok {
		return fmt.Errorf("cannot delete built-in template '%s'", name)
	}

	dir := TemplatesDir()
	path := filepath.Join(dir, name+".yaml")

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("template '%s' not found", name)
	}

	return os.Remove(path)
}

func (t *Template) Validate() error {
	tools := map[string]bool{
		"subfinder": true,
		"httpx":     true,
		"katana":    true,
		"gau":       true,
		"ffuf":      true,
	}

	if len(t.Steps) == 0 {
		return fmt.Errorf("template must have at least one step")
	}

	for _, step := range t.Steps {
		if step.Name == "" {
			return fmt.Errorf("step name is required")
		}
		if step.Tool == "" {
			return fmt.Errorf("step '%s' tool is required", step.Name)
		}
		if !tools[step.Tool] {
			return fmt.Errorf("step '%s' unknown tool: %s", step.Name, step.Tool)
		}
		if step.Order <= 0 {
			return fmt.Errorf("step '%s' order must be positive", step.Name)
		}
	}

	return nil
}
