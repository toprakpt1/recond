package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/recond/internal/daemon"
	"github.com/spf13/cobra"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Manage the recond daemon",
}

var daemonStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the recond daemon in background",
	Run: func(cmd *cobra.Command, args []string) {
		exe, err := os.Executable()
		if err != nil {
			exe = "recond"
		}

		daemonPath := filepath.Join(filepath.Dir(exe), "recond")

		if _, err := os.Stat(daemonPath); os.IsNotExist(err) {
			daemonPath = "recond"
		}

		daemonExe := exec.Command(daemonPath)
		daemonExe.Stdout = nil
		daemonExe.Stderr = nil

		if err := daemonExe.Start(); err != nil {
			fmt.Println("Error starting daemon:", err)
			return
		}

		fmt.Printf("Daemon started (PID: %d)\n", daemonExe.Process.Pid)
	},
}

var daemonStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the recond daemon",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Sending stop signal to daemon...")
		resp, err := daemon.SendCommand("daemon-status", nil)
		if err != nil {
			fmt.Println("Daemon is not running")
			return
		}
		_ = resp
		fmt.Println("Kill the daemon process manually:")
		fmt.Println("  pkill recond")
	},
}

var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check if daemon is running",
	Run: func(cmd *cobra.Command, args []string) {
		resp, err := daemon.SendCommand("daemon-status", nil)
		if err != nil {
			fmt.Println("Daemon is not running")
			return
		}
		if resp.Success {
			data := resp.Data.(map[string]interface{})
			fmt.Println("Daemon is running")
			fmt.Printf("  Data Directory: %s\n", data["data_dir"])
		}
	},
}

var daemonHealthCmd = &cobra.Command{
	Use:   "health",
	Short: "Show daemon health and tool status",
	Run: func(cmd *cobra.Command, args []string) {
		resp, err := daemon.SendCommand("health", nil)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		if !resp.Success {
			fmt.Println("Error:", resp.Error)
			return
		}

		data := resp.Data.(map[string]interface{})

		fmt.Println("Daemon Health:")
		fmt.Printf("  Running:      %v\n", data["running"])
		fmt.Printf("  Data Dir:     %s\n", data["data_dir"])
		fmt.Printf("  Socket:       %s\n", data["socket_path"])
		fmt.Printf("  Profile:      %s\n", data["profile"])

		if tools, ok := data["tool_status"].(map[string]interface{}); ok {
			fmt.Println("\nTool Status:")
			for tool, installed := range tools {
				status := "❌"
				if installed.(bool) {
					status = "✅"
				}
				fmt.Printf("  %s %s\n", status, tool)
			}
		}

		if activeJobs, ok := data["active_jobs"].([]interface{}); ok && len(activeJobs) > 0 {
			fmt.Printf("\nActive Jobs: %d\n", len(activeJobs))
			for _, j := range activeJobs {
				job := j.(map[string]interface{})
				fmt.Printf("  - %s (%s) [%s]\n", job["name"], job["id"], job["status"])
			}
		}
	},
}

func init() {
	daemonCmd.AddCommand(daemonStartCmd)
	daemonCmd.AddCommand(daemonStopCmd)
	daemonCmd.AddCommand(daemonStatusCmd)
	daemonCmd.AddCommand(daemonHealthCmd)
}
