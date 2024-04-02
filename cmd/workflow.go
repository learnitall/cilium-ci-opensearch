package cmd

import (
	"github.com/spf13/cobra"
)

var workflowCmd = &cobra.Command{
	Use:   "workflow",
	Short: "Scrape workflow runs for a repository",
}

func init() {
	rootCmd.AddCommand(workflowCmd)
}
