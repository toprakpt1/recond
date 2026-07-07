package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/recond/internal/daemon"
	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:   "export <job-id>",
	Short: "Export job results",
	Long:  `Export subdomains, alive hosts, URLs, or directories from a completed job.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		jobID := args[0]
		outputType, _ := cmd.Flags().GetString("type")
		format, _ := cmd.Flags().GetString("format")
		outputPath, _ := cmd.Flags().GetString("output")

		payload := struct {
			JobID  string `json:"job_id"`
			Type   string `json:"type"`
			Format string `json:"format"`
		}{
			JobID:  jobID,
			Type:   outputType,
			Format: format,
		}

		resp, err := daemon.SendCommand("export", payload)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		if !resp.Success {
			fmt.Println("Error:", resp.Error)
			return
		}

		data := resp.Data.(map[string]interface{})
		items, _ := data["items"].([]interface{})

		var output string
		switch format {
		case "json":
			result := map[string]interface{}{
				"items": items,
				"count": len(items),
				"type":  outputType,
			}
			jsonData, _ := json.MarshalIndent(result, "", "  ")
			output = string(jsonData)
		case "csv":
			var sb strings.Builder
			sb.WriteString("type,value,status\n")
			for _, item := range items {
				sb.WriteString(fmt.Sprintf("%s,%s,found\n", outputType, item))
			}
			output = sb.String()
		default:
			var lines []string
			for _, item := range items {
				lines = append(lines, fmt.Sprintf("%v", item))
			}
			output = strings.Join(lines, "\n")
		}

		if outputPath != "" {
			if err := os.WriteFile(outputPath, []byte(output+"\n"), 0644); err != nil {
				fmt.Printf("Error writing file: %v\n", err)
				return
			}
			fmt.Printf("Exported %d items to %s\n", len(items), outputPath)
		} else {
			fmt.Println(output)
		}
	},
}

var outputsCmd = &cobra.Command{
	Use:   "outputs <job-id>",
	Short: "List available outputs for a job",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		payload := struct {
			JobID string `json:"job_id"`
		}{JobID: args[0]}

		resp, err := daemon.SendCommand("outputs", payload)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		if !resp.Success {
			fmt.Println("Error:", resp.Error)
			return
		}

		data := resp.Data.(map[string]interface{})
		outputs, ok := data["outputs"].([]interface{})
		if !ok || len(outputs) == 0 {
			fmt.Println("No outputs found")
			return
		}

		fmt.Println("Available outputs:")
		for _, o := range outputs {
			fmt.Printf("  • %s\n", o)
		}
	},
}

func init() {
	exportCmd.Flags().StringP("type", "t", "all", "Output type (subdomains, alive, urls, directories, all)")
	exportCmd.Flags().StringP("format", "f", "text", "Output format (text, json, csv)")
	exportCmd.Flags().StringP("output", "o", "", "Output file path")
}
