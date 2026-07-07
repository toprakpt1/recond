package cli

import (
	"fmt"

	"github.com/toprakpt1/recond/internal/daemon"
	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop <job-id>",
	Short: "Stop a running or paused job",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		payload := daemon.ActionRequest{JobID: args[0]}

		resp, err := daemon.SendCommand("stop", payload)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		if !resp.Success {
			fmt.Println("Error:", resp.Error)
			return
		}

		fmt.Printf("Job %s stopped\n", args[0])
	},
}
