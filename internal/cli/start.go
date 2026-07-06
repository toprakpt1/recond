package cli

import (
	"encoding/json"
	"fmt"

	"github.com/recond/internal/daemon"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start <target>",
	Short: "Start a new recon job",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		target := args[0]
		profile, _ := cmd.Flags().GetString("profile")

		payload := daemon.StartRequest{
			Target:  target,
			Profile: profile,
		}

		resp, err := daemon.SendCommand("start", payload)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		if !resp.Success {
			fmt.Println("Error:", resp.Error)
			return
		}

		data := resp.Data.(map[string]interface{})
		fmt.Printf("Job created: %s\n", data["job_id"])
		fmt.Printf("  Target:  %s\n", data["target"])
		fmt.Printf("  Profile: %s\n", data["profile"])
		fmt.Printf("  Status:  %s\n", data["status"])
		fmt.Printf("  Steps:   %.0f\n", data["steps"])
	},
}

func init() {
	startCmd.Flags().StringP("profile", "p", "", "Resource profile (safe, balanced, aggressive)")
	_ = json.RawMessage{}
}
