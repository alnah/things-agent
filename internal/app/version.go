package app

import (
	"runtime/debug"

	buildinfo "github.com/alnah/things-agent/internal/build"
)

var readBuildInfo = debug.ReadBuildInfo

func effectiveCLIVersion() string {
	return buildinfo.EffectiveCLIVersionFrom(cliVersion, readBuildInfo)
}
