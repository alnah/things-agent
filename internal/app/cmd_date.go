package app

import (
	"time"

	commandlib "github.com/alnah/things-agent/internal/command"
	"github.com/spf13/cobra"
)

func newDateCmd() *cobra.Command {
	return commandlib.NewDateCmd(time.Now)
}

func formatCurrentDate(now time.Time) string {
	return commandlib.FormatCurrentDate(now)
}
