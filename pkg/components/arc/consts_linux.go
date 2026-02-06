//go:build linux
// +build linux

package arc

// Linux-specific Arc service names (systemd services)
var arcServices = []string{"himdsd", "gcarcservice", "extd"}
