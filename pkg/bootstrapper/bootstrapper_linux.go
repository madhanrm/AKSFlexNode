//go:build linux
// +build linux

package bootstrapper

import (
	"go.goms.io/aks/AKSFlexNode/pkg/components/arc"
	"go.goms.io/aks/AKSFlexNode/pkg/components/cni"
	"go.goms.io/aks/AKSFlexNode/pkg/components/containerd"
	"go.goms.io/aks/AKSFlexNode/pkg/components/kube_binaries"
	"go.goms.io/aks/AKSFlexNode/pkg/components/kubelet"
	"go.goms.io/aks/AKSFlexNode/pkg/components/npd"
	"go.goms.io/aks/AKSFlexNode/pkg/components/runc"
	"go.goms.io/aks/AKSFlexNode/pkg/components/services"
	"go.goms.io/aks/AKSFlexNode/pkg/components/system_configuration"
)

// getBootstrapSteps returns the ordered list of bootstrap steps for Linux
func (b *Bootstrapper) getBootstrapSteps() []Executor {
	return []Executor{
		arc.NewInstaller(b.logger),                  // Setup Arc
		services.NewUnInstaller(b.logger),           // Stop kubelet before setup
		system_configuration.NewInstaller(b.logger), // Configure system (early)
		runc.NewInstaller(b.logger),                 // Install runc (Linux container runtime)
		containerd.NewInstaller(b.logger),           // Install containerd
		kube_binaries.NewInstaller(b.logger),        // Install k8s binaries
		cni.NewInstaller(b.logger),                  // Setup CNI (after container runtime)
		kubelet.NewInstaller(b.logger),              // Configure kubelet service with Arc MSI auth
		npd.NewInstaller(b.logger),                  // Install Node Problem Detector
		services.NewInstaller(b.logger),             // Start services
	}
}

// getUnbootstrapSteps returns the ordered list of unbootstrap steps for Linux
func (b *Bootstrapper) getUnbootstrapSteps() []Executor {
	return []Executor{
		services.NewUnInstaller(b.logger),             // Stop services first
		npd.NewUnInstaller(b.logger),                  // Uninstall Node Problem Detector
		kubelet.NewUnInstaller(b.logger),              // Clean kubelet configuration
		cni.NewUnInstaller(b.logger),                  // Clean CNI configs
		kube_binaries.NewUnInstaller(b.logger),        // Uninstall k8s binaries
		containerd.NewUnInstaller(b.logger),           // Uninstall containerd binary
		runc.NewUnInstaller(b.logger),                 // Uninstall runc binary
		system_configuration.NewUnInstaller(b.logger), // Clean system settings
		arc.NewUnInstaller(b.logger),                  // Uninstall Arc (after cleanup)
	}
}
