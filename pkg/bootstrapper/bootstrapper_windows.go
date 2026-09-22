//go:build windows
// +build windows

package bootstrapper

import (
	"go.goms.io/aks/AKSFlexNode/pkg/components/arc"
	"go.goms.io/aks/AKSFlexNode/pkg/components/cni"
	"go.goms.io/aks/AKSFlexNode/pkg/components/containerd"
	"go.goms.io/aks/AKSFlexNode/pkg/components/kube_binaries"
	"go.goms.io/aks/AKSFlexNode/pkg/components/kubelet"
	"go.goms.io/aks/AKSFlexNode/pkg/components/runhcs"
	"go.goms.io/aks/AKSFlexNode/pkg/components/services"
	"go.goms.io/aks/AKSFlexNode/pkg/components/system_configuration"
)

// getBootstrapSteps returns the ordered list of bootstrap steps for Windows
func (b *Bootstrapper) getBootstrapSteps() []Executor {
	return []Executor{
		// Phase 1: System configuration (firewall, directories)
		system_configuration.NewInstaller(b.logger),

		// Phase 2: Container runtime
		containerd.NewInstaller(b.logger), // Install containerd (cross-platform)
		runhcs.NewInstaller(b.logger),     // Verify runhcs shim (bundled with containerd)

		// Phase 3: Kubernetes binaries
		kube_binaries.NewInstaller(b.logger), // Install kubelet, kubectl, kubeadm

		// Phase 4: Networking - Calico CNI for Windows (VXLAN backend)
		cni.NewInstaller(b.logger),

		// Phase 5: Kubelet configuration (Arc token, kubeconfig, service)
		kubelet.NewInstaller(b.logger),

		// Phase 6: Arc setup (cross-platform - uses Azure SDK + azcmagent)
		arc.NewInstaller(b.logger),

		// Phase 7: Services
		services.NewInstaller(b.logger), // Start services
	}
}

// getUnbootstrapSteps returns the ordered list of unbootstrap steps for Windows
func (b *Bootstrapper) getUnbootstrapSteps() []Executor {
	return []Executor{
		// Phase 1: Stop services
		services.NewUnInstaller(b.logger),

		// Phase 2: Arc cleanup (cross-platform - uses Azure SDK + azcmagent)
		arc.NewUnInstaller(b.logger),

		// Phase 3: Kubelet cleanup
		kubelet.NewUnInstaller(b.logger),

		// Phase 4: CNI cleanup - Calico for Windows
		cni.NewUnInstaller(b.logger),

		// Phase 5: K8s binaries cleanup
		kube_binaries.NewUnInstaller(b.logger),

		// Phase 6: Container runtime
		runhcs.NewUnInstaller(b.logger),     // Remove runhcs shim
		containerd.NewUnInstaller(b.logger), // Uninstall containerd

		// Phase 7: System cleanup
		system_configuration.NewUnInstaller(b.logger),
	}
}
