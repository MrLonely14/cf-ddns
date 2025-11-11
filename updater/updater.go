package updater

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/MrLonely14/cf-ddns/cloudflare"
	"github.com/MrLonely14/cf-ddns/config"
	"github.com/MrLonely14/cf-ddns/ipdetect"
)

// Updater manages DNS record updates
type Updater struct {
	cfg      *config.Config
	cfClient *cloudflare.Client
	detector *ipdetect.Detector
	state    *State
	mu       sync.RWMutex
}

// State tracks the last known IPs for each record
type State struct {
	Records map[string]string // key: "zoneID:name:type", value: last known IP
	mu      sync.RWMutex
}

// NewState creates a new state tracker
func NewState() *State {
	return &State{
		Records: make(map[string]string),
	}
}

// Get retrieves the last known IP for a record
func (s *State) Get(zoneID, name, recordType string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	key := fmt.Sprintf("%s:%s:%s", zoneID, name, recordType)
	return s.Records[key]
}

// Set stores the last known IP for a record
func (s *State) Set(zoneID, name, recordType, ip string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := fmt.Sprintf("%s:%s:%s", zoneID, name, recordType)
	s.Records[key] = ip
}

// NewUpdater creates a new DNS updater
func NewUpdater(cfg *config.Config, cfClient *cloudflare.Client, detector *ipdetect.Detector) *Updater {
	return &Updater{
		cfg:      cfg,
		cfClient: cfClient,
		detector: detector,
		state:    NewState(),
	}
}

// UpdateAll checks and updates all configured DNS records
func (u *Updater) UpdateAll(ctx context.Context) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(u.cfg.Records)*2) // max 2 types per record

	for _, record := range u.cfg.Records {
		for _, recordType := range record.Types {
			wg.Add(1)
			go func(rec config.DNSRecord, recType string) {
				defer wg.Done()
				if err := u.updateRecord(ctx, rec, recType); err != nil {
					errChan <- fmt.Errorf("failed to update %s (%s): %w", rec.Name, recType, err)
				}
			}(record, recordType)
		}
	}

	wg.Wait()
	close(errChan)

	// Collect all errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
		log.Printf("ERROR: %v", err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("encountered %d error(s) during update", len(errors))
	}

	return nil
}

// updateRecord updates a single DNS record if the IP has changed
func (u *Updater) updateRecord(ctx context.Context, record config.DNSRecord, recordType string) error {
	// Get current IP
	var currentIP string
	var err error

	if recordType == "A" {
		currentIP, err = u.detector.GetIPv4(ctx)
	} else if recordType == "AAAA" {
		currentIP, err = u.detector.GetIPv6(ctx)
	} else {
		return fmt.Errorf("invalid record type: %s", recordType)
	}

	if err != nil {
		return fmt.Errorf("failed to detect IP: %w", err)
	}

	// Check if IP has changed
	lastKnownIP := u.state.Get(record.ZoneID, record.Name, recordType)
	if currentIP == lastKnownIP && lastKnownIP != "" {
		log.Printf("No change for %s (%s): %s", record.Name, recordType, currentIP)
		return nil
	}

	// IP has changed or this is the first run, update DNS record
	log.Printf("Updating %s (%s): %s -> %s", record.Name, recordType, lastKnownIP, currentIP)

	err = u.cfClient.UpsertDNSRecord(
		ctx,
		record.ZoneID,
		record.Name,
		recordType,
		currentIP,
		record.TTL,
		record.Proxied,
	)
	if err != nil {
		return fmt.Errorf("failed to update Cloudflare DNS: %w", err)
	}

	// Update state
	u.state.Set(record.ZoneID, record.Name, recordType, currentIP)
	log.Printf("Successfully updated %s (%s) to %s", record.Name, recordType, currentIP)

	return nil
}

// InitializeState loads the current DNS records from Cloudflare to populate initial state
func (u *Updater) InitializeState(ctx context.Context) error {
	log.Println("Initializing state from Cloudflare...")

	for _, record := range u.cfg.Records {
		for _, recordType := range record.Types {
			existing, err := u.cfClient.GetDNSRecord(ctx, record.ZoneID, record.Name, recordType)
			if err != nil {
				// Record doesn't exist yet, skip
				log.Printf("Record %s (%s) not found in Cloudflare, will be created on first update", record.Name, recordType)
				continue
			}

			u.state.Set(record.ZoneID, record.Name, recordType, existing.Content)
			log.Printf("Loaded existing record: %s (%s) = %s", record.Name, recordType, existing.Content)
		}
	}

	log.Println("State initialization complete")
	return nil
}
