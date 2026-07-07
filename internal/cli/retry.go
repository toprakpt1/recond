package cli

import (
	"fmt"

	"github.com/recond/internal/daemon"
	"github.com/spf13/cobra"
)

var retryCmd = &cobra.Command{
	Use:   "retry <job-id>",
	Short: "Retry a failed or stopped job",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		profile, _ := cmd.Flags().GetString("profile")
		fromStep, _ := cmd.Flags().GetString("from-step")

		payload := struct {
			JobID    string `json:"job_id"`
			Profile  string `json:"profile,omitempty"`
			FromStep string `json:"from_step,omitempty"`
		}{
			JobID:    args[0],
			Profile:  profile,
			FromStep: fromStep,
		}

		resp, err := daemon.SendCommand("retry", payload)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		if !resp.Success {
			fmt.Println("Error:", resp.Error)
			return
		}

		data := resp.Data.(map[string]interface{})
		fmt.Printf("Job %s restarted (new ID: %s)\n", args[0], data["new_job_id"])
	},
}

var duplicateCmd = &cobra.Command{
	Use:   "duplicate <job-id>",
	Short: "Create a new job with the same target",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		profile, _ := cmd.Flags().GetString("profile")

		payload := struct {
			JobID   string `json:"job_id"`
			Profile string `json:"profile,omitempty"`
		}{
			JobID:   args[0],
			Profile: profile,
		}

		resp, err := daemon.SendCommand("duplicate", payload)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		if !resp.Success {
			fmt.Println("Error:", resp.Error)
			return
		}

		data := resp.Data.(map[string]interface{})
		fmt.Printf("New job created: %s (target: %s)\n", data["new_job_id"], data["target"])
	},
}

func init() {
	retryCmd.Flags().StringP("profile", "p", "", "Use a different profile")
	retryCmd.Flags().String("from-step", "", "Resume from a specific step")

	duplicateCmd.Flags().StringP("profile", "p", "", "Use a different profile")
}
