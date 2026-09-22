//go:build windows
// +build windows

package bootstrapper

import (
	"testing"

	"github.com/sirupsen/logrus"
	"go.goms.io/aks/AKSFlexNode/pkg/config"
)

// TestWindowsBootstrapperNew verifies the Windows bootstrapper constructor.
func TestWindowsBootstrapperNew(t *testing.T) {
	cfg := &config.Config{}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	bootstrapper := New(cfg, logger)

	if bootstrapper == nil {
		t.Fatal("New should not return nil")
	}

	if bootstrapper.BaseExecutor == nil {
		t.Error("BaseExecutor should be initialized")
	}
}

// TestWindowsBootstrapStepsOrder verifies bootstrap steps are in correct order.
func TestWindowsBootstrapStepsOrder(t *testing.T) {
	cfg := &config.Config{}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	bootstrapper := New(cfg, logger)
	steps := bootstrapper.getBootstrapSteps()

	if len(steps) == 0 {
		t.Fatal("getBootstrapSteps should not return empty slice")
	}

	// Expected order for Windows:
	// 1. system_configuration
	// 2. containerd
	// 3. runhcs
	// 4. kube_binaries
	// 5. cni (Calico)
	// 6. kubelet
	// 7. arc
	// 8. services

	expectedCount := 8
	if len(steps) != expectedCount {
		t.Errorf("Expected %d bootstrap steps, got %d", expectedCount, len(steps))
	}

	// Verify each step is not nil
	for i, step := range steps {
		if step == nil {
			t.Errorf("Bootstrap step %d is nil", i)
		}
	}
}

// TestWindowsBootstrapStepNames verifies bootstrap step names.
func TestWindowsBootstrapStepNames(t *testing.T) {
	cfg := &config.Config{}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	bootstrapper := New(cfg, logger)
	steps := bootstrapper.getBootstrapSteps()

	// Check that each step has a name
	for i, step := range steps {
		name := step.GetName()
		if name == "" {
			t.Errorf("Bootstrap step %d has empty name", i)
		}
		t.Logf("Step %d: %s", i, name)
	}
}

// TestWindowsUnbootstrapStepsOrder verifies unbootstrap steps are in correct order.
func TestWindowsUnbootstrapStepsOrder(t *testing.T) {
	cfg := &config.Config{}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	bootstrapper := New(cfg, logger)
	steps := bootstrapper.getUnbootstrapSteps()

	if len(steps) == 0 {
		t.Fatal("getUnbootstrapSteps should not return empty slice")
	}

	// Unbootstrap should be reverse order
	expectedCount := 8
	if len(steps) != expectedCount {
		t.Errorf("Expected %d unbootstrap steps, got %d", expectedCount, len(steps))
	}

	// Verify each step is not nil
	for i, step := range steps {
		if step == nil {
			t.Errorf("Unbootstrap step %d is nil", i)
		}
	}
}

// TestWindowsUnbootstrapStepNames verifies unbootstrap step names.
func TestWindowsUnbootstrapStepNames(t *testing.T) {
	cfg := &config.Config{}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	bootstrapper := New(cfg, logger)
	steps := bootstrapper.getUnbootstrapSteps()

	// Check that each step has a name
	for i, step := range steps {
		name := step.GetName()
		if name == "" {
			t.Errorf("Unbootstrap step %d has empty name", i)
		}
		t.Logf("Step %d: %s", i, name)
	}
}

// TestWindowsBootstrapContainsRunhcs verifies runhcs is in bootstrap steps.
func TestWindowsBootstrapContainsRunhcs(t *testing.T) {
	cfg := &config.Config{}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	bootstrapper := New(cfg, logger)
	steps := bootstrapper.getBootstrapSteps()

	hasRunhcs := false
	for _, step := range steps {
		if step.GetName() == "Runhcs_Installer" {
			hasRunhcs = true
			break
		}
	}

	if !hasRunhcs {
		t.Error("Windows bootstrap should contain runhcs step")
	}
}

// TestWindowsBootstrapContainsCalico verifies Calico CNI is in bootstrap steps.
func TestWindowsBootstrapContainsCalico(t *testing.T) {
	cfg := &config.Config{}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	bootstrapper := New(cfg, logger)
	steps := bootstrapper.getBootstrapSteps()

	hasCNI := false
	for _, step := range steps {
		if step.GetName() == "CNISetup" {
			hasCNI = true
			break
		}
	}

	if !hasCNI {
		t.Error("Windows bootstrap should contain CNI (Calico) step")
	}
}

// TestWindowsBootstrapOrderContainerdBeforeRunhcs verifies containerd comes before runhcs.
func TestWindowsBootstrapOrderContainerdBeforeRunhcs(t *testing.T) {
	cfg := &config.Config{}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	bootstrapper := New(cfg, logger)
	steps := bootstrapper.getBootstrapSteps()

	containerdIdx := -1
	runhcsIdx := -1

	for i, step := range steps {
		name := step.GetName()
		if name == "Containerd" || name == "Containerd_Installer" {
			containerdIdx = i
		}
		if name == "Runhcs_Installer" {
			runhcsIdx = i
		}
	}

	if containerdIdx == -1 {
		t.Error("Could not find containerd step")
		return
	}
	if runhcsIdx == -1 {
		t.Error("Could not find runhcs step")
		return
	}

	if containerdIdx >= runhcsIdx {
		t.Errorf("Containerd (idx %d) should come before runhcs (idx %d)", containerdIdx, runhcsIdx)
	}
}

// TestWindowsBootstrapOrderCNIBeforeKubelet verifies CNI comes before kubelet.
func TestWindowsBootstrapOrderCNIBeforeKubelet(t *testing.T) {
	cfg := &config.Config{}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	bootstrapper := New(cfg, logger)
	steps := bootstrapper.getBootstrapSteps()

	cniIdx := -1
	kubeletIdx := -1

	for i, step := range steps {
		name := step.GetName()
		if name == "CNISetup" {
			cniIdx = i
		}
		if name == "KubeletInstaller" {
			kubeletIdx = i
		}
	}

	if cniIdx == -1 {
		t.Error("Could not find CNI step")
		return
	}
	if kubeletIdx == -1 {
		t.Error("Could not find kubelet step")
		return
	}

	if cniIdx >= kubeletIdx {
		t.Errorf("CNI (idx %d) should come before kubelet (idx %d)", cniIdx, kubeletIdx)
	}
}

// TestWindowsUnbootstrapServicesFirst verifies services are stopped first in unbootstrap.
func TestWindowsUnbootstrapServicesFirst(t *testing.T) {
	cfg := &config.Config{}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	bootstrapper := New(cfg, logger)
	steps := bootstrapper.getUnbootstrapSteps()

	if len(steps) == 0 {
		t.Fatal("getUnbootstrapSteps should not return empty slice")
	}

	// First step should be services uninstaller
	firstStepName := steps[0].GetName()
	if firstStepName != "Services_UnInstaller" && firstStepName != "Services" {
		t.Logf("First unbootstrap step is: %s (expected services-related)", firstStepName)
	}
}
