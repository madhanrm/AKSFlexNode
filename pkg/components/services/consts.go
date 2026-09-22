package services

import "time"

const (
	// Service names
	ContainerdService = "containerd"
	KubeletService    = "kubelet"
	NPDService        = "node-problem-detector"

	// Service startup timeout
	ServiceStartupTimeout = 30 * time.Second
)
