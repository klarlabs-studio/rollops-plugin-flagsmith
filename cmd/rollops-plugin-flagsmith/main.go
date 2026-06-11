// Command rollops-plugin-flagsmith is a Rollops feature-flag provider plugin
// backed by Flagsmith. Build it, pin its sha256, and point a rollout's
// featureFlags.plugin at the binary.
package main

import (
	"fmt"
	"os"

	flagsmith "github.com/klarlabs-studio/rollops-plugin-flagsmith"
	"go.klarlabs.de/rollops/pkg/plugin"
)

// version is overwritten at build time via -ldflags.
var version = "dev"

func main() {
	safety := plugin.Safety{
		// The plugin reaches the Flagsmith API; declare it so the host policy
		// can allow-list egress. Operators set the concrete host via the
		// FLAGSMITH_API_URL env and their pluginhost policy.
		NetworkHosts: []string{"api.flagsmith.com:443"},
		EnvVars:      []string{"FLAGSMITH_API_URL", "FLAGSMITH_TOKEN", "FLAGSMITH_PROJECT_ID"},
		RiskClass:    plugin.RiskActive,
	}
	if err := plugin.ServeFlagProvider("klarlabs/flagsmith", version, flagsmith.FromEnv(), safety); err != nil {
		fmt.Fprintln(os.Stderr, "rollops-plugin-flagsmith:", err)
		os.Exit(1)
	}
}
