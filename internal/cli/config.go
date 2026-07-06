package cli

import (
	"fmt"
	"os"
	"path/filepath"

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
		fmt.Printf("Data Directory: %s\n", cfg.DataDir)
		fmt.Printf("Socket Path:    %s\n", cfg.SocketPath)
		fmt.Printf("Default Profile: %s\n", cfg.DefaultProfile)
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key=value>",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Use config file at:", config.ConfigPath())
		fmt.Println("Edit it directly or use environment variables (RECOND_*)")
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
		home, _ := os.UserHomeDir()
		path := filepath.Join(home, ".recond", "config.yaml")
		fmt.Println("Default config created at:", path)
	},
}

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configInitCmd)
}
