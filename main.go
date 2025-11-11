package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/MrLonely14/cf-ddns/cloudflare"
	"github.com/MrLonely14/cf-ddns/config"
	"github.com/MrLonely14/cf-ddns/installer"
	"github.com/MrLonely14/cf-ddns/ipdetect"
	"github.com/MrLonely14/cf-ddns/updater"
)

const version = "1.0.0"

func main() {
	// Define commands and flags
	runCmd := flag.NewFlagSet("run", flag.ExitOnError)
	installCmd := flag.NewFlagSet("install", flag.ExitOnError)
	uninstallCmd := flag.NewFlagSet("uninstall", flag.ExitOnError)
	statusCmd := flag.NewFlagSet("status", flag.ExitOnError)

	// Flags for run command
	configPath := runCmd.String("config", "config.yaml", "Path to configuration file")

	// Flags for install command
	installConfigPath := installCmd.String("config", "/etc/cf-ddns/config.yaml", "Path to configuration file")
	installUser := installCmd.String("user", os.Getenv("USER"), "User to run the service as")

	// Parse command
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "run":
		runCmd.Parse(os.Args[2:])
		runDaemon(*configPath)
	case "install":
		installCmd.Parse(os.Args[2:])
		installService(*installConfigPath, *installUser)
	case "uninstall":
		uninstallCmd.Parse(os.Args[2:])
		uninstallService()
	case "status":
		statusCmd.Parse(os.Args[2:])
		checkStatus()
	case "version", "-v", "--version":
		fmt.Printf("cf-ddns version %s\n", version)
	case "help", "-h", "--help":
		printUsage()
	default:
		// Default to run command if no subcommand specified
		runDaemon("config.yaml")
	}
}

func printUsage() {
	fmt.Println("Cloudflare Dynamic DNS Updater")
	fmt.Println("\nUsage:")
	fmt.Println("  cf-ddns run [flags]          Run the daemon (default)")
	fmt.Println("  cf-ddns install [flags]      Install as system service")
	fmt.Println("  cf-ddns uninstall            Uninstall system service")
	fmt.Println("  cf-ddns status               Check service status")
	fmt.Println("  cf-ddns version              Show version")
	fmt.Println("  cf-ddns help                 Show this help message")
	fmt.Println("\nRun Flags:")
	fmt.Println("  -config string    Path to configuration file (default \"config.yaml\")")
	fmt.Println("\nInstall Flags:")
	fmt.Println("  -config string    Path to configuration file (default \"/etc/cf-ddns/config.yaml\")")
	fmt.Println("  -user string      User to run the service as (default: current user)")
}

func runDaemon(configPath string) {
	log.Printf("Starting Cloudflare DDNS Updater v%s", version)

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	log.Printf("Loaded configuration from %s", configPath)
	log.Printf("Check interval: %s", cfg.CheckInterval)
	log.Printf("Monitoring %d DNS record(s)", len(cfg.Records))

	// Create Cloudflare client
	cfClient, err := cloudflare.NewClient(cfg.Cloudflare.APIToken)
	if err != nil {
		log.Fatalf("Failed to create Cloudflare client: %v", err)
	}

	// Create IP detector
	detector := ipdetect.NewDetector()

	// Create updater
	upd := updater.NewUpdater(cfg, cfClient, detector)

	// Initialize state from existing DNS records
	ctx := context.Background()
	if err := upd.InitializeState(ctx); err != nil {
		log.Printf("Warning: Failed to initialize state: %v", err)
	}

	// Run initial update
	log.Println("Running initial DNS update...")
	if err := upd.UpdateAll(ctx); err != nil {
		log.Printf("Initial update completed with errors: %v", err)
	} else {
		log.Println("Initial update completed successfully")
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start daemon loop
	ticker := time.NewTicker(cfg.GetCheckInterval())
	defer ticker.Stop()

	log.Println("Daemon started, waiting for IP changes...")

	for {
		select {
		case <-ticker.C:
			log.Println("Checking for IP changes...")
			if err := upd.UpdateAll(ctx); err != nil {
				log.Printf("Update failed: %v", err)
			}
		case sig := <-sigChan:
			log.Printf("Received signal %v, shutting down gracefully...", sig)
			log.Println("Performing final DNS update before shutdown...")
			if err := upd.UpdateAll(ctx); err != nil {
				log.Printf("Final update failed: %v", err)
			}
			log.Println("Shutdown complete")
			return
		}
	}
}

func installService(configPath, user string) {
	log.Println("Installing cf-ddns as system service...")

	// Get executable path
	exePath, err := os.Executable()
	if err != nil {
		log.Fatalf("Failed to get executable path: %v", err)
	}

	// Install service
	if err := installer.Install(exePath, configPath, user); err != nil {
		log.Fatalf("Failed to install service: %v", err)
	}

	log.Println("Service installed successfully!")
	log.Println("\nNext steps:")
	log.Printf("1. Edit the example configuration file:")
	log.Printf("   Example: %s/config.example.yaml", filepath.Dir(configPath))
	log.Printf("   Copy it to: %s", configPath)
	log.Printf("   Command: sudo cp %s/config.example.yaml %s", filepath.Dir(configPath), configPath)
	log.Println("2. Edit the config file with your Cloudflare API token and zones")
	log.Println("3. Start the service:")
	installer.PrintStartCommand()
}

func uninstallService() {
	log.Println("Uninstalling cf-ddns system service...")

	if err := installer.Uninstall(); err != nil {
		log.Fatalf("Failed to uninstall service: %v", err)
	}

	log.Println("Service uninstalled successfully!")
}

func checkStatus() {
	status, err := installer.Status()
	if err != nil {
		log.Fatalf("Failed to check status: %v", err)
	}

	fmt.Println(status)
}
