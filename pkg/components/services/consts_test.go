package services

import (
	"testing"
	"time"
)

func TestServicesConstants(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{"ContainerdService", ContainerdService, "containerd"},
		{"KubeletService", KubeletService, "kubelet"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("%s = %s, want %s", tt.name, tt.value, tt.expected)
			}
		})
	}
}

func TestServiceStartupTimeout(t *testing.T) {
	expected := 30 * time.Second
	if ServiceStartupTimeout != expected {
		t.Errorf("ServiceStartupTimeout = %v, want %v", ServiceStartupTimeout, expected)
	}
}
