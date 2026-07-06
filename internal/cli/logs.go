package cli

import (
	"fmt"

	"github.com/recond/internal/daemon"
	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs <job-id>",
	Short: "Show job logs",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		level, _ := cmd.Flags().GetString("level")
		limit, _ := cmd.Flags().GetInt("limit")

		payload := struct {
			JobID string `json:"job_id"`
			Level string `json:"level,omitempty"`
			Limit int    `json:"limit"`
		}{
			JobID: args[0],
			Level: level,
			Limit: limit,
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

		for _, l := range logs {
			logEntry := l.(map[string]interface{})
			fmt.Printf("[%s] [%s] %s\n", logEntry["created_at"], logEntry["level"], logEntry["message"])
		}
	},
}

func init() {
	logsCmd.Flags().StringP("level", "l", "", "Filter by level (info, warn, error, debug)")
	logsCmd.Flags().IntP("limit", "n", 100, "Number of log entries to show")
}
