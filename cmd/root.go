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

const (
	timeFormatYearMonthDayHour = "2006-01-02T15"
	timeFormatYearMonthDay     = "2006-01-02"
)

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
