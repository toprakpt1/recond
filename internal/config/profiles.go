package config

import "time"

type Profile struct {
	Name        string        `json:"name"`
	Concurrency int           `json:"concurrency"`
	RateLimit   int           `json:"rate_limit"`
	CPUMax      int           `json:"cpu_max"`
	RAMMax      string        `json:"ram_max"`
	Timeout     time.Duration `json:"timeout"`
}

var DefaultProfiles = map[string]Profile{
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

func GetProfile(name string) (Profile, bool) {
	p, ok := DefaultProfiles[name]
	return p, ok
}
