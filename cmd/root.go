package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

type typeRootParams struct {
	Index   string
	Verbose bool
}

var (
	rootCmd    = &cobra.Command{}
	rootParams = &typeRootParams{}
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&rootParams.Index, "index", "i", "runs", "OpenSearch index to target")
	rootCmd.PersistentFlags().BoolVarP(&rootParams.Verbose, "verbose", "v", false, "Enable debug logging")

}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func addTimeFilterParams(cmd *cobra.Command) {

}
