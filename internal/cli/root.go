package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/toprakpt1/recond/internal/version"
)

var rootCmd = &cobra.Command{
	Use:   "recon",
	Short: "Recon orchestrator - manage recon jobs",
	Long: `recond is a recon job orchestrator that manages reconnaissance
workflows. It runs as a daemon and provides a CLI interface to
start, pause, resume, and monitor recon jobs.

Examples:
  recon start example.com
  recon status job-abc123
  recon pause job-abc123
  recon resume job-abc123
  recon list`,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	SilenceUsage: true,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version.String())
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(pauseCmd)
	rootCmd.AddCommand(resumeCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(daemonCmd)
	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(outputsCmd)
	rootCmd.AddCommand(templatesCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(deleteAllCmd)
	rootCmd.AddCommand(retryCmd)
	rootCmd.AddCommand(duplicateCmd)
}
