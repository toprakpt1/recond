package cli

import (
	"fmt"

	"github.com/toprakpt1/recond/internal/daemon"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <job-id>",
	Short: "Delete a job and its outputs",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		force, _ := cmd.Flags().GetBool("force")

		if !force {
			fmt.Printf("Are you sure you want to delete job %s? (y/N): ", args[0])
			var confirm string
			fmt.Scanln(&confirm)
			if confirm != "y" && confirm != "Y" {
				fmt.Println("Cancelled")
				return
			}
		}

		payload := struct {
			JobID string `json:"job_id"`
		}{JobID: args[0]}

		resp, err := daemon.SendCommand("delete", payload)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		if !resp.Success {
			fmt.Println("Error:", resp.Error)
			return
		}

		fmt.Printf("Job %s deleted\n", args[0])
	},
}

var deleteAllCmd = &cobra.Command{
	Use:   "delete-all",
	Short: "Delete all completed jobs",
	Run: func(cmd *cobra.Command, args []string) {
		force, _ := cmd.Flags().GetBool("force")

		if !force {
			fmt.Print("Are you sure you want to delete ALL completed jobs? (y/N): ")
			var confirm string
			fmt.Scanln(&confirm)
			if confirm != "y" && confirm != "Y" {
				fmt.Println("Cancelled")
				return
			}
		}

		resp, err := daemon.SendCommand("delete-all", nil)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		if !resp.Success {
			fmt.Println("Error:", resp.Error)
			return
		}

		data := resp.Data.(map[string]interface{})
		count := data["deleted"].(float64)
		fmt.Printf("Deleted %d jobs\n", int(count))
	},
}

func init() {
	deleteCmd.Flags().BoolP("force", "f", false, "Skip confirmation")
	deleteAllCmd.Flags().BoolP("force", "f", false, "Skip confirmation")
}
