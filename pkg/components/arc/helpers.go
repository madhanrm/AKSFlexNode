package arc

import (
	"os/exec"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/hybridcompute/armhybridcompute"
)

// isArcAgentInstalled checks if azcmagent is available in PATH
// This works on both Linux and Windows
func isArcAgentInstalled() bool {
	_, err := exec.LookPath("azcmagent")
	return err == nil
}

func getArcMachineIdentityID(arcMachine *armhybridcompute.Machine) string {
	if arcMachine != nil &&
		arcMachine.Identity != nil &&
		arcMachine.Identity.PrincipalID != nil {
		return *arcMachine.Identity.PrincipalID
	}
	return ""
}
