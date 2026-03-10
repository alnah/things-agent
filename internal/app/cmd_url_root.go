package app

import (
	commandlib "github.com/alnah/things-agent/internal/command"
	"github.com/spf13/cobra"
)

func newURLCmd() *cobra.Command {
	return commandlib.NewURLRootCmd(
		newURLAddCmd(),
		newURLUpdateCmd(),
		newURLAddProjectCmd(),
		newURLUpdateProjectCmd(),
		newURLShowCmd(),
		newURLSearchCmd(),
		newURLVersionCmd(),
		newURLJSONCmd(),
	)
}
