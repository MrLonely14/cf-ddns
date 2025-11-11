package ipdetect

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// Detector handles IP address detection
type Detector struct {
	client     *http.Client
	ipv4Cache  string
	ipv6Cache  string
	lastUpdate time.Time
}

// NewDetector creates a new IP detector
func NewDetector() *Detector {
	return &Detector{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// IPv4 services to try in order
var ipv4Services = []string{
	"https://api.ipify.org",
	"https://icanhazip.com",
	"https://ifconfig.me/ip",
	"https://checkip.amazonaws.com",
}

// IPv6 services to try in order
var ipv6Services = []string{
	"https://api64.ipify.org",
	"https://ipv6.icanhazip.com",
	"https://v6.ident.me",
}

// GetIPv4 detects the current public IPv4 address
func (d *Detector) GetIPv4(ctx context.Context) (string, error) {
	for _, service := range ipv4Services {
		ip, err := d.fetchIP(ctx, service, false)
		if err == nil && ip != "" {
			d.ipv4Cache = ip
			d.lastUpdate = time.Now()
			return ip, nil
		}
	}
	return "", fmt.Errorf("failed to detect IPv4 address from all services")
}

// GetIPv6 detects the current public IPv6 address
func (d *Detector) GetIPv6(ctx context.Context) (string, error) {
	for _, service := range ipv6Services {
		ip, err := d.fetchIP(ctx, service, true)
		if err == nil && ip != "" {
			d.ipv6Cache = ip
			d.lastUpdate = time.Now()
			return ip, nil
		}
	}
	return "", fmt.Errorf("failed to detect IPv6 address from all services")
}

// fetchIP fetches IP from a service and validates it
func (d *Detector) fetchIP(ctx context.Context, url string, isIPv6 bool) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	// For IPv6, prefer IPv6 transport
	if isIPv6 {
		transport := &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return (&net.Dialer{
					Timeout: 5 * time.Second,
				}).DialContext(ctx, "tcp6", addr)
			},
		}
		client := &http.Client{
			Timeout:   10 * time.Second,
			Transport: transport,
		}
		resp, err := client.Do(req)
		if err != nil {
			// Fallback to default client
			resp, err = d.client.Do(req)
			if err != nil {
				return "", err
			}
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("service returned status %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}

		ip := strings.TrimSpace(string(body))

		// Validate IP address
		parsedIP := net.ParseIP(ip)
		if parsedIP == nil {
			return "", fmt.Errorf("invalid IP address: %s", ip)
		}

		// Check if it's the correct IP type
		if isIPv6 && parsedIP.To4() != nil {
			return "", fmt.Errorf("expected IPv6 but got IPv4")
		}
		if !isIPv6 && parsedIP.To4() == nil {
			return "", fmt.Errorf("expected IPv4 but got IPv6")
		}

		return ip, nil
	}

	// For IPv4, use default client
	resp, err := d.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("service returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	ip := strings.TrimSpace(string(body))

	// Validate IP address
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return "", fmt.Errorf("invalid IP address: %s", ip)
	}

	// Check if it's IPv4
	if parsedIP.To4() == nil {
		return "", fmt.Errorf("expected IPv4 but got IPv6")
	}

	return ip, nil
}

// GetCachedIPv4 returns the last cached IPv4 address
func (d *Detector) GetCachedIPv4() string {
	return d.ipv4Cache
}

// GetCachedIPv6 returns the last cached IPv6 address
func (d *Detector) GetCachedIPv6() string {
	return d.ipv6Cache
}

// HasIPChanged checks if IP has changed since last check
func (d *Detector) HasIPChanged(ctx context.Context, recordType string, lastKnownIP string) (bool, string, error) {
	var currentIP string
	var err error

	if recordType == "A" {
		currentIP, err = d.GetIPv4(ctx)
	} else if recordType == "AAAA" {
		currentIP, err = d.GetIPv6(ctx)
	} else {
		return false, "", fmt.Errorf("invalid record type: %s", recordType)
	}

	if err != nil {
		return false, "", err
	}

	return currentIP != lastKnownIP, currentIP, nil
}
