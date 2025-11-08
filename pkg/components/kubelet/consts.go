package kubelet

const (
	// System directories
	EtcDefaultDir       = "/etc/default"
	KubeletServiceDir   = "/etc/systemd/system/kubelet.service.d"

	// Configuration file paths
	KubeletDefaultsPath     = "/etc/default/kubelet"
	KubeletServicePath      = "/etc/systemd/system/kubelet.service"
	KubeletContainerdConfig = "/etc/systemd/system/kubelet.service.d/10-containerd.conf"

	// Runtime configuration paths
	KubeletConfigPath          = "/var/lib/kubelet/config.yaml"
	KubeletKubeConfig          = "/etc/kubernetes/kubelet.conf"
	KubeletBootstrapKubeConfig = "/etc/kubernetes/bootstrap-kubelet.conf"
)