package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Profile struct {
	Name        string        `json:"name"`
	Concurrency int           `json:"concurrency"`
	RateLimit   int           `json:"rate_limit"`
	CPUMax      int           `json:"cpu_max"`
	RAMMax      string        `json:"ram_max"`
	Timeout     time.Duration `json:"timeout"`
}

func DefaultProfiles() map[string]Profile {
	return map[string]Profile{
		"safe": {
			Name:        "safe",
			Concurrency: 3,
			RateLimit:   10,
			CPUMax:      20,
			RAMMax:      "1GB",
			Timeout:     30 * time.Second,
		},
		"balanced": {
			Name:        "balanced",
			Concurrency: 10,
			RateLimit:   50,
			CPUMax:      50,
			RAMMax:      "2GB",
			Timeout:     15 * time.Second,
		},
		"aggressive": {
			Name:        "aggressive",
			Concurrency: 25,
			RateLimit:   100,
			CPUMax:      80,
			RAMMax:      "4GB",
			Timeout:     10 * time.Second,
		},
	}
}

func LoadProfiles() map[string]Profile {
	if v == nil {
		return DefaultProfiles()
	}

	profiles := DefaultProfiles()

	if v.IsSet("profiles") {
		profilesMap := v.GetStringMap("profiles")
		for name := range profilesMap {
			if existing, ok := profiles[name]; ok {
				profiles[name] = loadProfileFromConfig(name, existing)
			} else {
				profiles[name] = loadProfileFromConfig(name, Profile{Name: name})
			}
		}
	}

	return profiles
}

func loadProfileFromConfig(name string, p Profile) Profile {
	key := "profiles." + name

	if v.IsSet(key+".concurrency") {
		p.Concurrency = v.GetInt(key + ".concurrency")
	}
	if v.IsSet(key+".rate_limit") {
		p.RateLimit = v.GetInt(key + ".rate_limit")
	}
	if v.IsSet(key+".cpu_max") {
		p.CPUMax = v.GetInt(key + ".cpu_max")
	}
	if v.IsSet(key+".ram_max") {
		p.RAMMax = v.GetString(key + ".ram_max")
	}
	if v.IsSet(key+".timeout") {
		p.Timeout = v.GetDuration(key + ".timeout")
	}

	p.Name = name
	return p
}

func GetProfile(name string) (Profile, bool) {
	profiles := LoadProfiles()
	p, ok := profiles[name]
	return p, ok
}

func ListProfiles() []Profile {
	profiles := LoadProfiles()
	var result []Profile
	for _, p := range profiles {
		result = append(result, p)
	}
	return result
}

func CreateProfile(name string, p Profile) error {
	if v == nil {
		return fmt.Errorf("config not initialized")
	}

	key := "profiles." + name
	if v.IsSet(key) {
		return fmt.Errorf("profile '%s' already exists", name)
	}

	v.Set(key+".concurrency", p.Concurrency)
	v.Set(key+".rate_limit", p.RateLimit)
	v.Set(key+".cpu_max", p.CPUMax)
	v.Set(key+".ram_max", p.RAMMax)
	v.Set(key+".timeout", p.Timeout.String())

	return v.WriteConfigAs(ConfigPath())
}

func UpdateProfile(name string, p Profile) error {
	if v == nil {
		return fmt.Errorf("config not initialized")
	}

	key := "profiles." + name

	v.Set(key+".concurrency", p.Concurrency)
	v.Set(key+".rate_limit", p.RateLimit)
	v.Set(key+".cpu_max", p.CPUMax)
	v.Set(key+".ram_max", p.RAMMax)
	v.Set(key+".timeout", p.Timeout.String())

	return v.WriteConfigAs(ConfigPath())
}

func DeleteProfile(name string) error {
	if v == nil {
		return fmt.Errorf("config not initialized")
	}

	if name == "safe" || name == "balanced" || name == "aggressive" {
		return fmt.Errorf("cannot delete built-in profile '%s'", name)
	}

	key := "profiles." + name
	if !v.IsSet(key) {
		return fmt.Errorf("profile '%s' not found", name)
	}

	v.Set(key, nil)

	configPath := ConfigPath()
	if err := v.WriteConfigAs(configPath); err != nil {
		return err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil
	}

	content := string(data)
	content = strings.Replace(content, "  "+name+":\n", "", -1)
	content = strings.Replace(content, "    "+name+":\n", "", -1)

	return os.WriteFile(configPath, []byte(content), 0644)
}

func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh", int(d.Hours()))
}

func ParseRAMMax(ram string) (int64, error) {
	ram = strings.ToUpper(strings.TrimSpace(ram))

	multiplier := int64(1)
	numStr := ram

	if strings.HasSuffix(ram, "GB") {
		multiplier = 1024 * 1024 * 1024
		numStr = strings.TrimSuffix(ram, "GB")
	} else if strings.HasSuffix(ram, "MB") {
		multiplier = 1024 * 1024
		numStr = strings.TrimSuffix(ram, "MB")
	} else if strings.HasSuffix(ram, "KB") {
		multiplier = 1024
		numStr = strings.TrimSuffix(ram, "KB")
	}

	num, err := strconv.ParseFloat(strings.TrimSpace(numStr), 64)
	if err != nil {
		return 0, fmt.Errorf("invalid RAM value: %s", ram)
	}

	return int64(num * float64(multiplier)), nil
}
