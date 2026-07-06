package cli

import (
	"fmt"

	"github.com/recond/internal/daemon"
	"github.com/spf13/cobra"
)

var pauseCmd = &cobra.Command{
	Use:   "pause <job-id>",
	Short: "Pause a running job",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		payload := daemon.ActionRequest{JobID: args[0]}

		resp, err := daemon.SendCommand("pause", payload)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		if !resp.Success {
			fmt.Println("Error:", resp.Error)
			return
		}

		fmt.Printf("Job %s paused\n", args[0])
	},
}
