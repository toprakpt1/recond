package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/toprakpt1/recond/internal/daemon"
	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs <job-id>",
	Short: "Show job logs",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		level, _ := cmd.Flags().GetString("level")
		limit, _ := cmd.Flags().GetInt("limit")
		step, _ := cmd.Flags().GetString("step")
		search, _ := cmd.Flags().GetString("search")
		follow, _ := cmd.Flags().GetBool("follow")
		export, _ := cmd.Flags().GetString("export")

		if follow {
			runFollowMode(args[0], level, step, search)
			return
		}

		payload := struct {
			JobID  string `json:"job_id"`
			Level  string `json:"level,omitempty"`
			Step   string `json:"step,omitempty"`
			Search string `json:"search,omitempty"`
			Limit  int    `json:"limit"`
		}{
			JobID:  args[0],
			Level:  level,
			Step:   step,
			Search: search,
			Limit:  limit,
		}

		resp, err := daemon.SendCommand("logs", payload)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		if !resp.Success {
			fmt.Println("Error:", resp.Error)
			return
		}

		data := resp.Data.(map[string]interface{})
		logs := data["logs"].([]interface{})

		if len(logs) == 0 {
			fmt.Println("No logs found")
			return
		}

		if export != "" {
			exportLogs(logs, export)
			return
		}

		for _, l := range logs {
			logEntry := l.(map[string]interface{})
			fmt.Printf("[%s] [%s] %s\n", logEntry["created_at"], logEntry["level"], logEntry["message"])
		}
	},
}

func runFollowMode(jobID, level, step, search string) {
	fmt.Printf("Following logs for job %s (Ctrl+C to stop)...\n\n", jobID)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	var lastID float64 = 0

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-sigCh:
			fmt.Println("\nStopped following logs")
			return
		case <-ticker.C:
			payload := struct {
				JobID  string  `json:"job_id"`
				Level  string  `json:"level,omitempty"`
				Step   string  `json:"step,omitempty"`
				Search string  `json:"search,omitempty"`
				Limit  int     `json:"limit"`
				After  float64 `json:"after,omitempty"`
			}{
				JobID:  jobID,
				Level:  level,
				Step:   step,
				Search: search,
				Limit:  50,
				After:  lastID,
			}

			resp, err := daemon.SendCommand("logs", payload)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				continue
			}

			if !resp.Success {
				continue
			}

			data := resp.Data.(map[string]interface{})
			logs, ok := data["logs"].([]interface{})
			if !ok || len(logs) == 0 {
				continue
			}

			for _, l := range logs {
				logEntry := l.(map[string]interface{})
				id := logEntry["id"].(float64)
				if id > lastID {
					lastID = id
					fmt.Printf("[%s] [%s] %s\n",
						logEntry["created_at"], logEntry["level"], logEntry["message"])
				}
			}
		}
	}
}

func exportLogs(logs []interface{}, format string) {
	switch format {
	case "json":
		data, _ := json.MarshalIndent(logs, "", "  ")
		fmt.Println(string(data))
	case "csv":
		fmt.Println("id,job_id,step_id,level,message,created_at")
		for _, l := range logs {
			entry := l.(map[string]interface{})
			fmt.Printf("%v,%v,%v,%v,\"%v\",%v\n",
				entry["id"], entry["job_id"], entry["step_id"],
				entry["level"], entry["message"], entry["created_at"])
		}
	default:
		fmt.Printf("Unknown export format: %s (use json or csv)\n", format)
	}
}

func init() {
	logsCmd.Flags().StringP("level", "l", "", "Filter by level (info, warn, error, debug)")
	logsCmd.Flags().IntP("limit", "n", 100, "Number of log entries to show")
	logsCmd.Flags().String("step", "", "Filter by step name")
	logsCmd.Flags().String("search", "", "Search in log messages")
	logsCmd.Flags().BoolP("follow", "f", false, "Follow logs in real-time")
	logsCmd.Flags().String("export", "", "Export logs (json or csv)")

	_ = strings.TrimSpace
}
