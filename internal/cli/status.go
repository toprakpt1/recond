package cli

import (
	"fmt"

	"github.com/recond/internal/daemon"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status <job-id>",
	Short: "Show job status and progress",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		payload := daemon.ActionRequest{JobID: args[0]}

		resp, err := daemon.SendCommand("status", payload)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		if !resp.Success {
			fmt.Println("Error:", resp.Error)
			return
		}

		data := resp.Data.(map[string]interface{})

		jobData := data["job"].(map[string]interface{})
		stepsData := data["steps"].([]interface{})
		currentStep := data["current_step"].(string)
		overallProgress := data["overall_progress"].(float64)

		fmt.Printf("Job:     %s\n", jobData["name"])
		fmt.Printf("ID:      %s\n", jobData["id"])
		fmt.Printf("Status:  %s\n", jobData["status"])
		fmt.Printf("Target:  %s\n", jobData["target"])
		fmt.Printf("Profile: %s\n", jobData["profile"])
		fmt.Printf("Progress: %.0f%%\n", overallProgress)

		if currentStep != "" {
			fmt.Printf("Current Step: %s\n", currentStep)
		}

		fmt.Println("\nSteps:")
		for _, s := range stepsData {
			step := s.(map[string]interface{})
			status := step["status"].(string)
			progress := step["progress"].(float64)
			icon := "⬜"
			switch status {
			case "completed":
				icon = "✅"
			case "running":
				icon = "▶️ "
			case "failed":
				icon = "❌"
			case "paused":
				icon = "⏸️ "
			}
			fmt.Printf("  %s %s [%s] (%.0f%%)\n", icon, step["name"], status, progress)
		}
	},
}
