package cli

import (
	"runtime/debug"
	"strings"
)

const develVersion = "(devel)"

// ResolveVersion returns the version shown to users and embedded into generated
// site metadata.
func ResolveVersion() string {
	return resolveVersion(Version, buildInfoMainVersion())
}

func resolveVersion(injected, buildInfoVersion string) string {
	if v := strings.TrimSpace(injected); v != "" && v != "dev" {
		return v
	}
	if v := strings.TrimSpace(buildInfoVersion); v != "" && v != develVersion {
		return v
	}
	return "dev"
}

func buildInfoMainVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok || info == nil {
		return ""
	}
	return info.Main.Version
}
