//go:build windows
// +build windows

package arc

import (
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

// isArcServicesRunning checks if Arc services are running using Windows SCM
func isArcServicesRunning() bool {
	if !isArcAgentInstalled() {
		return false
	}

	// Connect to Windows Service Control Manager
	m, err := mgr.Connect()
	if err != nil {
		return false
	}
	defer m.Disconnect()

	// Check each Arc service
	for _, serviceName := range arcServices {
		service, err := m.OpenService(serviceName)
		if err != nil {
			return false
		}

		status, err := service.Query()
		service.Close()
		if err != nil {
			return false
		}

		// Check if service is running
		if status.State != svc.Running {
			return false
		}
	}

	return true
}
