// Package platform provides OS-specific abstractions for AKS Flex Node.
// It enables the same codebase to work on both Linux and Windows worker nodes.
package platform

import (
	"context"
	"runtime"
)

// OS represents the operating system type
type OS string

const (
	Linux   OS = "linux"
	Windows OS = "windows"
)

// Platform provides OS-specific operations for AKS Flex Node
type Platform interface {
	// OS returns the operating system type
	OS() OS

	// Paths returns OS-specific paths configuration
	Paths() *PathConfig

	// Service returns the service manager for this platform
	Service() ServiceManager

	// Command returns the command executor for this platform
	Command() CommandExecutor

	// FileSystem returns the filesystem operations for this platform
	FileSystem() FileSystem
}

// CommandExecutor provides OS-specific command execution
type CommandExecutor interface {
	// Run executes a command and waits for completion
	Run(ctx context.Context, name string, args ...string) error

	// RunWithOutput executes a command and returns output
	RunWithOutput(ctx context.Context, name string, args ...string) (string, error)

	// RunPrivileged executes a command with elevated privileges (sudo on Linux, admin on Windows)
	RunPrivileged(ctx context.Context, name string, args ...string) error

	// RunPrivilegedWithOutput executes a privileged command and returns output
	RunPrivilegedWithOutput(ctx context.Context, name string, args ...string) (string, error)
}

// FileSystem provides OS-specific filesystem operations
type FileSystem interface {
	// FileExists checks if a file exists
	FileExists(path string) bool

	// DirectoryExists checks if a directory exists
	DirectoryExists(path string) bool

	// CreateDirectory creates a directory with appropriate permissions
	CreateDirectory(path string) error

	// WriteFile writes content to a file atomically
	WriteFile(path string, content []byte, perm uint32) error

	// ReadFile reads content from a file
	ReadFile(path string) ([]byte, error)

	// RemoveFile removes a file
	RemoveFile(path string) error

	// RemoveDirectory removes a directory recursively
	RemoveDirectory(path string) error

	// DownloadFile downloads a file from URL to destination
	DownloadFile(url, destination string) error

	// ExtractTarGz extracts a tar.gz file to destination
	ExtractTarGz(archive, destination string) error

	// GetArchitecture returns the system architecture (amd64, arm64)
	GetArchitecture() (string, error)
}

// ServiceManager provides OS-specific service management
type ServiceManager interface {
	// Install installs a service with the given configuration
	Install(config *ServiceConfig) error

	// Uninstall removes a service
	Uninstall(name string) error

	// Start starts a service
	Start(name string) error

	// Stop stops a service
	Stop(name string) error

	// Restart restarts a service
	Restart(name string) error

	// Enable enables a service to start on boot
	Enable(name string) error

	// Disable disables a service from starting on boot
	Disable(name string) error

	// IsActive checks if a service is running
	IsActive(name string) bool

	// IsEnabled checks if a service is enabled
	IsEnabled(name string) bool

	// Exists checks if a service exists
	Exists(name string) bool

	// WaitForService waits for a service to become active
	WaitForService(name string, timeoutSeconds int) error

	// ReloadDaemon reloads the service manager configuration (e.g., systemctl daemon-reload)
	ReloadDaemon() error
}

// ServiceConfig contains configuration for installing a service
type ServiceConfig struct {
	Name           string            // Service name
	DisplayName    string            // Human-readable display name
	Description    string            // Service description
	BinaryPath     string            // Path to the executable
	Args           []string          // Command line arguments
	WorkingDir     string            // Working directory
	Dependencies   []string          // Services this depends on
	Environment    map[string]string // Environment variables
	User           string            // User to run as (Linux)
	RestartPolicy  RestartPolicy     // Restart behavior
	RestartDelayMs int               // Delay between restarts in milliseconds
}

// RestartPolicy defines service restart behavior
type RestartPolicy string

const (
	RestartAlways    RestartPolicy = "always"
	RestartOnFailure RestartPolicy = "on-failure"
	RestartNever     RestartPolicy = "never"
)

// currentPlatform holds the singleton platform instance
var currentPlatform Platform

// Current returns the platform instance for the current OS
func Current() Platform {
	if currentPlatform == nil {
		currentPlatform = newPlatform()
	}
	return currentPlatform
}

// IsLinux returns true if running on Linux
func IsLinux() bool {
	return runtime.GOOS == "linux"
}

// IsWindows returns true if running on Windows
func IsWindows() bool {
	return runtime.GOOS == "windows"
}
