// Package netpolicy applies the panel's process-local network safety policy.
package netpolicy

import (
	"os"
	"strings"
)

func safeGoDebug(value string) string {
	settings := make([]string, 0)
	for setting := range strings.SplitSeq(value, ",") {
		key, _, _ := strings.Cut(setting, "=")
		if setting != "" && key != "multipathtcp" {
			settings = append(settings, setting)
		}
	}
	return strings.Join(append(settings, "multipathtcp=0"), ",")
}

// Enforce disables Go's implicit MPTCP listeners before any panel/setup/ACME
// listener is opened. Runtime enforcement also overrides inherited GODEBUG.
// This mitigates the demonstrated CVE-2026-98432 exploit, not the kernel bug.
func Enforce() error {
	return os.Setenv("GODEBUG", safeGoDebug(os.Getenv("GODEBUG")))
}

// Environment applies the same policy to child cores, including old versions
// and rollbacks. Other environment variables and Go debug settings are kept.
func Environment(environ []string) []string {
	result := make([]string, 0, len(environ)+1)
	var debug string
	for _, entry := range environ {
		if value, ok := strings.CutPrefix(entry, "GODEBUG="); ok {
			debug = value // Match exec.Cmd's last-value-wins environment semantics.
		} else {
			result = append(result, entry)
		}
	}
	return append(result, "GODEBUG="+safeGoDebug(debug))
}
