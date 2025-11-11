package installer

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
)

//go:embed templates/cf-ddns.service
var systemdTemplate string

//go:embed templates/cf-ddns.plist
var launchdTemplate string

//go:embed templates/install.ps1
var windowsTemplate string

//go:embed templates/config.example.yaml
var configExample string

// ServiceConfig holds the configuration for service installation
type ServiceConfig struct {
	ExecPath   string
	ConfigPath string
	ConfigDir  string
	User       string
}

// createExampleConfig creates a config.example.yaml file in the config directory
func createExampleConfig(configPath string) error {
	configDir := filepath.Dir(configPath)
	examplePath := filepath.Join(configDir, "config.example.yaml")

	// Create directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write example config
	if err := os.WriteFile(examplePath, []byte(configExample), 0644); err != nil {
		return fmt.Errorf("failed to write example config: %w", err)
	}

	return nil
}

// Install installs the service for the current operating system
func Install(execPath, configPath, user string) error {
	// Create example config file
	if err := createExampleConfig(configPath); err != nil {
		return fmt.Errorf("failed to create example config: %w", err)
	}

	switch runtime.GOOS {
	case "linux":
		return installLinux(execPath, configPath, user)
	case "darwin":
		return installMacOS(execPath, configPath, user)
	case "windows":
		return installWindows(execPath, configPath, user)
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// Uninstall removes the service for the current operating system
func Uninstall() error {
	switch runtime.GOOS {
	case "linux":
		return uninstallLinux()
	case "darwin":
		return uninstallMacOS()
	case "windows":
		return uninstallWindows()
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// Status checks the status of the service
func Status() (string, error) {
	switch runtime.GOOS {
	case "linux":
		return statusLinux()
	case "darwin":
		return statusMacOS()
	case "windows":
		return statusWindows()
	default:
		return "", fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// PrintStartCommand prints the command to start the service
func PrintStartCommand() {
	switch runtime.GOOS {
	case "linux":
		fmt.Println("   sudo systemctl start cf-ddns")
		fmt.Println("   sudo systemctl enable cf-ddns")
		fmt.Println("\nView logs:")
		fmt.Println("   sudo journalctl -u cf-ddns -f")
	case "darwin":
		fmt.Println("   launchctl load ~/Library/LaunchAgents/com.cf-ddns.plist")
		fmt.Println("\nView logs:")
		fmt.Println("   tail -f /tmp/cf-ddns.log")
	case "windows":
		fmt.Println("   Start-ScheduledTask -TaskName \"CloudflareDDNS\"")
		fmt.Println("\nView in Task Scheduler:")
		fmt.Println("   taskschd.msc")
	}
}

// installLinux installs the systemd service
func installLinux(execPath, configPath, user string) error {
	serviceFile := "/etc/systemd/system/cf-ddns.service"

	// Parse and execute template
	tmpl, err := template.New("systemd").Parse(systemdTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	cfg := ServiceConfig{
		ExecPath:   execPath,
		ConfigPath: configPath,
		ConfigDir:  filepath.Dir(configPath),
		User:       user,
	}

	// Create temporary file
	tmpFile, err := os.CreateTemp("", "cf-ddns-*.service")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if err := tmpl.Execute(tmpFile, cfg); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}
	tmpFile.Close()

	// Copy to systemd directory (requires sudo)
	cmd := exec.Command("sudo", "cp", tmpFile.Name(), serviceFile)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to copy service file: %w\n%s", err, output)
	}

	// Set proper permissions
	cmd = exec.Command("sudo", "chmod", "644", serviceFile)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set permissions: %w\n%s", err, output)
	}

	// Reload systemd
	cmd = exec.Command("sudo", "systemctl", "daemon-reload")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to reload systemd: %w\n%s", err, output)
	}

	return nil
}

// uninstallLinux removes the systemd service
func uninstallLinux() error {
	// Stop service
	exec.Command("sudo", "systemctl", "stop", "cf-ddns").Run()

	// Disable service
	exec.Command("sudo", "systemctl", "disable", "cf-ddns").Run()

	// Remove service file
	cmd := exec.Command("sudo", "rm", "-f", "/etc/systemd/system/cf-ddns.service")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to remove service file: %w\n%s", err, output)
	}

	// Reload systemd
	cmd = exec.Command("sudo", "systemctl", "daemon-reload")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to reload systemd: %w\n%s", err, output)
	}

	return nil
}

// statusLinux checks the systemd service status
func statusLinux() (string, error) {
	cmd := exec.Command("systemctl", "status", "cf-ddns")
	output, _ := cmd.CombinedOutput()
	return string(output), nil
}

// installMacOS installs the launchd service
func installMacOS(execPath, configPath, user string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	plistPath := filepath.Join(homeDir, "Library", "LaunchAgents", "com.cf-ddns.plist")

	// Create LaunchAgents directory if it doesn't exist
	agentsDir := filepath.Dir(plistPath)
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		return fmt.Errorf("failed to create LaunchAgents directory: %w", err)
	}

	// Parse and execute template
	tmpl, err := template.New("launchd").Parse(launchdTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	cfg := ServiceConfig{
		ExecPath:   execPath,
		ConfigPath: configPath,
	}

	// Create plist file
	file, err := os.Create(plistPath)
	if err != nil {
		return fmt.Errorf("failed to create plist file: %w", err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, cfg); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	// Load the service
	cmd := exec.Command("launchctl", "load", plistPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to load service: %w\n%s", err, output)
	}

	return nil
}

// uninstallMacOS removes the launchd service
func uninstallMacOS() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	plistPath := filepath.Join(homeDir, "Library", "LaunchAgents", "com.cf-ddns.plist")

	// Unload the service
	cmd := exec.Command("launchctl", "unload", plistPath)
	cmd.Run() // Ignore errors if service is not loaded

	// Remove plist file
	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove plist file: %w", err)
	}

	return nil
}

// statusMacOS checks the launchd service status
func statusMacOS() (string, error) {
	cmd := exec.Command("launchctl", "list", "com.cf-ddns")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "Service is not running", nil
	}
	return string(output), nil
}

// installWindows installs the Windows scheduled task
func installWindows(execPath, configPath, user string) error {
	// Parse and execute template
	tmpl, err := template.New("windows").Parse(windowsTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	cfg := ServiceConfig{
		ExecPath:   execPath,
		ConfigPath: configPath,
	}

	// Create temporary PowerShell script
	tmpFile, err := os.CreateTemp("", "cf-ddns-install-*.ps1")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if err := tmpl.Execute(tmpFile, cfg); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}
	tmpFile.Close()

	// Execute PowerShell script
	cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-File", tmpFile.Name())
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to execute install script: %w\n%s", err, output)
	}

	return nil
}

// uninstallWindows removes the Windows scheduled task
func uninstallWindows() error {
	cmd := exec.Command("schtasks", "/Delete", "/TN", "CloudflareDDNS", "/F")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to delete scheduled task: %w\n%s", err, output)
	}

	return nil
}

// statusWindows checks the Windows scheduled task status
func statusWindows() (string, error) {
	cmd := exec.Command("schtasks", "/Query", "/TN", "CloudflareDDNS", "/FO", "LIST", "/V")
	output, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(output), "cannot find") {
			return "Service is not installed", nil
		}
		return "", fmt.Errorf("failed to query scheduled task: %w", err)
	}

	return string(output), nil
}
