//go:build linux
// +build linux

package arc

import (
	"os/exec"

	"go.goms.io/aks/AKSFlexNode/pkg/utils"
)

// isArcServicesRunning checks if Arc services are running using systemd
func isArcServicesRunning() bool {
	if !isArcAgentInstalled() {
		return false
	}

	for _, service := range arcServices {
		if !utils.IsServiceActive(service) {
			return false
		}
	}

	// Also check if azcmagent process is running
	cmd := exec.Command("pgrep", "-f", "azcmagent")
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}
