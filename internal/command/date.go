package command

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

func NewDateCmd(now func() time.Time) *cobra.Command {
	return &cobra.Command{
		Use:   "date",
		Short: "Show current weekday, date, time, and timezone",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(cmd.OutOrStdout(), FormatCurrentDate(now()))
		},
	}
}

func FormatCurrentDate(now time.Time) string {
	return now.Format("Monday 2006-01-02 15:04:05 MST")
}
