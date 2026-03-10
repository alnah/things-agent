package app

import (
	commandlib "github.com/alnah/things-agent/internal/command"
	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	return commandlib.NewVersionCmd(effectiveCLIVersion)
}
