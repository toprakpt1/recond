package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/toprakpt1/recond/internal/daemon"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all jobs",
	Run: func(cmd *cobra.Command, args []string) {
		statusFilter, _ := cmd.Flags().GetString("status")

		var payload interface{}
		if statusFilter != "" {
			payload = struct {
				Status string `json:"status"`
			}{Status: statusFilter}
		}

		resp, err := daemon.SendCommand("list", payload)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		if !resp.Success {
			fmt.Println("Error:", resp.Error)
			return
		}

		data := resp.Data.(map[string]interface{})
		jobsRaw, ok := data["jobs"].([]interface{})
		if !ok || jobsRaw == nil {
			fmt.Println("No jobs found")
			return
		}

		if len(jobsRaw) == 0 {
			fmt.Println("No jobs found")
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tSTATUS\tPROFILE\tCREATED")

		for _, j := range jobsRaw {
			job, ok := j.(map[string]interface{})
			if !ok {
				continue
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				job["id"], job["name"], job["status"], job["profile"], job["created_at"])
		}
		w.Flush()
	},
}

func init() {
	listCmd.Flags().StringP("status", "s", "", "Filter by status (running, paused, completed, failed)")
}
