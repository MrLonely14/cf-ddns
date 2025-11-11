package cloudflare

import (
	"context"
	"fmt"

	"github.com/cloudflare/cloudflare-go"
)

// Client wraps the Cloudflare API client
type Client struct {
	api *cloudflare.API
}

// DNSRecordInfo holds information about a DNS record
type DNSRecordInfo struct {
	ID      string
	ZoneID  string
	Name    string
	Type    string
	Content string
	TTL     int
	Proxied bool
}

// NewClient creates a new Cloudflare client
func NewClient(apiToken string) (*Client, error) {
	if apiToken == "" {
		return nil, fmt.Errorf("API token is required")
	}

	api, err := cloudflare.NewWithAPIToken(apiToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create Cloudflare client: %w", err)
	}

	return &Client{
		api: api,
	}, nil
}

// GetDNSRecord finds a DNS record by zone ID, name, and type
func (c *Client) GetDNSRecord(ctx context.Context, zoneID, name, recordType string) (*DNSRecordInfo, error) {
	// Create resource container for the zone
	rc := cloudflare.ZoneIdentifier(zoneID)

	// List DNS records with filters
	records, _, err := c.api.ListDNSRecords(ctx, rc, cloudflare.ListDNSRecordsParams{
		Name: name,
		Type: recordType,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list DNS records: %w", err)
	}

	// Check if we found any records
	if len(records) == 0 {
		return nil, fmt.Errorf("DNS record not found: %s (%s)", name, recordType)
	}

	// Return the first matching record
	record := records[0]
	return &DNSRecordInfo{
		ID:      record.ID,
		ZoneID:  zoneID,
		Name:    record.Name,
		Type:    record.Type,
		Content: record.Content,
		TTL:     record.TTL,
		Proxied: *record.Proxied,
	}, nil
}

// UpdateDNSRecord updates an existing DNS record
func (c *Client) UpdateDNSRecord(ctx context.Context, recordID, zoneID, name, recordType, content string, ttl int, proxied bool) error {
	// Create resource container for the zone
	rc := cloudflare.ZoneIdentifier(zoneID)

	_, err := c.api.UpdateDNSRecord(ctx, rc, cloudflare.UpdateDNSRecordParams{
		ID:      recordID,
		Content: content,
		TTL:     ttl,
		Proxied: &proxied,
	})
	if err != nil {
		return fmt.Errorf("failed to update DNS record: %w", err)
	}

	return nil
}

// CreateDNSRecord creates a new DNS record if it doesn't exist
func (c *Client) CreateDNSRecord(ctx context.Context, zoneID, name, recordType, content string, ttl int, proxied bool) (*DNSRecordInfo, error) {
	// Create resource container for the zone
	rc := cloudflare.ZoneIdentifier(zoneID)

	record, err := c.api.CreateDNSRecord(ctx, rc, cloudflare.CreateDNSRecordParams{
		Name:    name,
		Type:    recordType,
		Content: content,
		TTL:     ttl,
		Proxied: &proxied,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create DNS record: %w", err)
	}

	return &DNSRecordInfo{
		ID:      record.ID,
		ZoneID:  zoneID,
		Name:    record.Name,
		Type:    record.Type,
		Content: record.Content,
		TTL:     record.TTL,
		Proxied: *record.Proxied,
	}, nil
}

// UpsertDNSRecord updates a DNS record if it exists, or creates it if it doesn't
func (c *Client) UpsertDNSRecord(ctx context.Context, zoneID, name, recordType, content string, ttl int, proxied bool) error {
	// Try to get existing record
	existing, err := c.GetDNSRecord(ctx, zoneID, name, recordType)
	if err != nil {
		// Record doesn't exist, create it
		_, err := c.CreateDNSRecord(ctx, zoneID, name, recordType, content, ttl, proxied)
		return err
	}

	// Record exists, update it
	return c.UpdateDNSRecord(ctx, existing.ID, zoneID, name, recordType, content, ttl, proxied)
}
