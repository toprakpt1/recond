package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/recond/internal/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadConfig()
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		fmt.Printf("Data Directory:  %s\n", cfg.DataDir)
		fmt.Printf("Socket Path:     %s\n", cfg.SocketPath)
		fmt.Printf("Default Profile: %s\n", cfg.DefaultProfile)
		fmt.Printf("Max Retries:     %d\n", cfg.MaxRetries)
		fmt.Printf("Retry Backoff:   %s\n", cfg.RetryBackoff)
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		key, value := args[0], args[1]

		if err := config.Init(); err != nil {
			fmt.Println("Error initializing config:", err)
			return
		}

		switch key {
		case "default_profile":
			_, ok := config.GetProfile(value)
			if !ok {
				fmt.Printf("Error: profile '%s' not found\n", value)
				return
			}
		case "max_retries":
			if _, err := strconv.Atoi(value); err != nil {
				fmt.Println("Error: max_retries must be a number")
				return
			}
		}

		if err := config.Set(key, value); err != nil {
			fmt.Println("Error:", err)
			return
		}

		fmt.Printf("Set %s = %s\n", key, value)
	},
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize default configuration",
	Run: func(cmd *cobra.Command, args []string) {
		if err := config.WriteDefaultConfig(); err != nil {
			fmt.Println("Error:", err)
			return
		}
		fmt.Println("Default config created at:", config.ConfigPath())
	},
}

var configProfilesCmd = &cobra.Command{
	Use:   "profiles",
	Short: "Manage profiles",
}

var configProfilesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all profiles",
	Run: func(cmd *cobra.Command, args []string) {
		config.Init()
		profiles := config.ListProfiles()

		if len(profiles) == 0 {
			fmt.Println("No profiles found")
			return
		}

		fmt.Printf("%-15s %-12s %-12s %-8s %-8s %-10s\n",
			"NAME", "CONCURRENCY", "RATE_LIMIT", "CPU_MAX", "RAM_MAX", "TIMEOUT")
		fmt.Println(strings.Repeat("-", 70))

		for _, p := range profiles {
			fmt.Printf("%-15s %-12d %-12d %-8d %-8s %-10s\n",
				p.Name, p.Concurrency, p.RateLimit, p.CPUMax, p.RAMMax, p.Timeout)
		}
	},
}

var configProfilesShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show profile details",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		config.Init()
		p, ok := config.GetProfile(args[0])
		if !ok {
			fmt.Printf("Profile '%s' not found\n", args[0])
			return
		}

		fmt.Printf("Profile: %s\n", p.Name)
		fmt.Printf("  Concurrency: %d\n", p.Concurrency)
		fmt.Printf("  Rate Limit:  %d req/s\n", p.RateLimit)
		fmt.Printf("  CPU Max:     %d%%\n", p.CPUMax)
		fmt.Printf("  RAM Max:     %s\n", p.RAMMax)
		fmt.Printf("  Timeout:     %s\n", p.Timeout)
	},
}

var configProfilesCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new profile",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		config.Init()

		concurrency, _ := cmd.Flags().GetInt("concurrency")
		rateLimit, _ := cmd.Flags().GetInt("rate-limit")
		cpuMax, _ := cmd.Flags().GetInt("cpu-max")
		ramMax, _ := cmd.Flags().GetString("ram-max")
		timeout, _ := cmd.Flags().GetString("timeout")

		p := config.Profile{
			Name:        name,
			Concurrency: concurrency,
			RateLimit:   rateLimit,
			CPUMax:      cpuMax,
			RAMMax:      ramMax,
		}

		if t, err := parseTimeout(timeout); err == nil {
			p.Timeout = t
		} else {
			fmt.Printf("Error: invalid timeout: %v\n", err)
			return
		}

		if err := config.CreateProfile(name, p); err != nil {
			fmt.Println("Error:", err)
			return
		}

		fmt.Printf("Profile '%s' created\n", name)
	},
}

var configProfilesDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a profile",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		config.Init()
		if err := config.DeleteProfile(args[0]); err != nil {
			fmt.Println("Error:", err)
			return
		}
		fmt.Printf("Profile '%s' deleted\n", args[0])
	},
}

func parseTimeout(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "s") {
		v, err := strconv.Atoi(strings.TrimSuffix(s, "s"))
		if err != nil {
			return 0, err
		}
		return time.Duration(v) * time.Second, nil
	}
	if strings.HasSuffix(s, "m") {
		v, err := strconv.Atoi(strings.TrimSuffix(s, "m"))
		if err != nil {
			return 0, err
		}
		return time.Duration(v) * time.Minute, nil
	}
	if strings.HasSuffix(s, "h") {
		v, err := strconv.Atoi(strings.TrimSuffix(s, "h"))
		if err != nil {
			return 0, err
		}
		return time.Duration(v) * time.Hour, nil
	}

	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid duration format: %s", s)
	}
	return time.Duration(v) * time.Second, nil
}

func init() {
	configProfilesCreateCmd.Flags().Int("concurrency", 10, "Max concurrent operations")
	configProfilesCreateCmd.Flags().Int("rate-limit", 50, "Max requests per second")
	configProfilesCreateCmd.Flags().Int("cpu-max", 50, "Max CPU usage percentage")
	configProfilesCreateCmd.Flags().String("ram-max", "2GB", "Max RAM usage")
	configProfilesCreateCmd.Flags().String("timeout", "15s", "Step timeout")

	configProfilesCmd.AddCommand(configProfilesListCmd)
	configProfilesCmd.AddCommand(configProfilesShowCmd)
	configProfilesCmd.AddCommand(configProfilesCreateCmd)
	configProfilesCmd.AddCommand(configProfilesDeleteCmd)

	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configProfilesCmd)

	_ = os.ExpandEnv
}
