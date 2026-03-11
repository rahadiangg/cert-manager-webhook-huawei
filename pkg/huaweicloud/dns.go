package huaweicloud

import (
	"fmt"
	"strings"
	"time"

	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/region"
	dns "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/dns/v2"
	dnsregion "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/dns/v2/region"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/dns/v2/model"
)

// DNSClient is a wrapper around Huawei Cloud DNS SDK client
type DNSClient struct {
	client     *dns.DnsClient
	regionID   string
	projectID  string
	zoneName   string
	zoneID     string
}

// OperationContext for DNS operations
type OperationContext struct {
	UID     string
	Action  string
	DNSName string
	Issuer  string
}

// NewDNSClient creates a new Huawei Cloud DNS client
func NewDNSClient(regionName, projectID, ak, sk, zoneName string) (*DNSClient, error) {
	Debug("creating DNS client",
		"region", regionName,
		"project_id", projectID,
		"zone_name", zoneName,
	)

	// Create auth credential
	auth, err := basic.NewCredentialsBuilder().
		WithAk(ak).
		WithSk(sk).
		SafeBuild()
	if err != nil {
		return nil, fmt.Errorf("failed to create credentials: %w", err)
	}

	// Create region
	dnsRegion, err := getRegionID(regionName)
	if err != nil {
		Error("invalid region", "region", regionName, "error", err)
		return nil, fmt.Errorf("invalid region %s: %w", regionName, err)
	}

	// Create HTTP client
	httpClient, err := dns.DnsClientBuilder().
		WithRegion(dnsRegion).
		WithCredential(auth).
		SafeBuild()
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	// Create DNS client
	client := dns.NewDnsClient(httpClient)

	d := &DNSClient{
		client:    client,
		regionID:  dnsRegion.Id,
		projectID: projectID,
		zoneName:  zoneName,
	}

	// Get zone ID
	zoneID, err := d.getZoneID()
	if err != nil {
		Error("failed to get zone ID", "zone_name", zoneName, "error", err)
		return nil, fmt.Errorf("failed to get zone ID: %w", err)
	}
	d.zoneID = zoneID

	Info("DNS client created successfully",
		"region", regionName,
		"zone_id", zoneID,
		"zone_name", zoneName,
	)

	return d, nil
}

// getRegionID converts region name to region using SDK's region lookup
func getRegionID(region string) (*region.Region, error) {
	// Use SDK's built-in region lookup for DNS service
	dnsRegion, err := dnsregion.SafeValueOf(region)
	if err != nil {
		return nil, fmt.Errorf("invalid region %s: %w", region, err)
	}
	return dnsRegion, nil
}

// getZoneID retrieves the zone ID from zone name
func (d *DNSClient) getZoneID() (string, error) {
	// List all public zones
	request := &model.ListPublicZonesRequest{}
	zoneType := "public"
	request.Type = &zoneType
	limit := int32(100)
	request.Limit = &limit

	response, err := d.client.ListPublicZones(request)
	if err != nil {
		return "", fmt.Errorf("failed to list zones: %w", err)
	}

	if response.Zones == nil || len(*response.Zones) == 0 {
		return "", fmt.Errorf("no zones found")
	}

	// Find matching zone
	for _, zone := range *response.Zones {
		// Zone names may or may not have trailing dot, handle both
		zoneName := strings.TrimSuffix(*zone.Name, ".")
		if zoneName == d.zoneName || zoneName == d.zoneName+"." {
			return *zone.Id, nil
		}
	}

	return "", fmt.Errorf("zone %s not found", d.zoneName)
}

// CreateTXTRecord creates a TXT record for ACME challenge
// This method is idempotent and handles race conditions
func (d *DNSClient) CreateTXTRecord(ctx OperationContext, fqdn, value string, ttl int) error {
	Info("creating TXT record",
		"uid", ctx.UID,
		"action", ctx.Action,
		"dns_name", ctx.DNSName,
		"record_name", fqdn,
		"step", "initialize",
	)

	recordName := d.extractRecordName(fqdn)
	const maxRetries = 3

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			Debug("retrying TXT record creation",
				"uid", ctx.UID,
				"action", ctx.Action,
				"dns_name", ctx.DNSName,
				"attempt", attempt+1,
				"max_retries", maxRetries,
				"record_name", recordName)
			time.Sleep(time.Duration(1<<uint(attempt)) * time.Second)
		}

		// Step 1: List existing records
		Info("listing existing TXT records",
			"uid", ctx.UID,
			"action", ctx.Action,
			"dns_name", ctx.DNSName,
			"record_name", recordName,
			"step", "list_records",
		)

		existingRecords, err := d.listTXTRecords(recordName)
		if err != nil {
			Warn("failed to list existing records, will retry",
				"uid", ctx.UID,
				"action", ctx.Action,
				"dns_name", ctx.DNSName,
				"record_name", recordName,
				"error", err,
				"attempt", attempt+1)
			continue
		}

		// Step 2: Handle based on existing record count
		switch len(existingRecords) {
		case 0:
			// No existing record - create new one
			Info("no existing records found, creating new record",
				"uid", ctx.UID,
				"action", ctx.Action,
				"dns_name", ctx.DNSName,
				"record_name", recordName,
				"step", "create",
			)

			err := d.createTXTRecordWithRetry(recordName, value, ttl)
			if err != nil {
				// Check if it was created by another goroutine
				records, checkErr := d.listTXTRecords(recordName)
				if checkErr == nil && len(records) > 0 {
					for _, r := range records {
						if r.Records != nil && len(*r.Records) > 0 {
							storedValue := strings.Trim((*r.Records)[0], `"`)
							if storedValue == value {
								Info("record was created by concurrent process (idempotent)",
									"uid", ctx.UID,
									"action", ctx.Action,
									"dns_name", ctx.DNSName,
									"record_name", recordName,
									"step", "complete")
								return nil
							}
						}
					}
				}
				Warn("create failed, will retry",
					"uid", ctx.UID,
					"action", ctx.Action,
					"dns_name", ctx.DNSName,
					"record_name", recordName,
					"error", err,
					"attempt", attempt+1)
				continue
			}
			Info("created new TXT record successfully",
				"uid", ctx.UID,
				"action", ctx.Action,
				"dns_name", ctx.DNSName,
				"record_name", recordName,
				"value", value,
				"step", "complete")
			return nil

		case 1:
			// One existing record - check if it matches
			record := existingRecords[0]
			if record.Records != nil && len(*record.Records) > 0 {
				storedValue := strings.Trim((*record.Records)[0], `"`)

				if storedValue == value {
					Info("record already exists with correct value (idempotent)",
						"uid", ctx.UID,
						"action", ctx.Action,
						"dns_name", ctx.DNSName,
						"record_name", recordName,
						"step", "complete")
					return nil
				}

				// Value mismatch - update existing record
				Info("updating existing record with new value",
					"uid", ctx.UID,
					"action", ctx.Action,
					"dns_name", ctx.DNSName,
					"record_name", recordName,
					"old_value", storedValue,
					"new_value", value,
					"step", "update",
				)

				quotedValue := fmt.Sprintf("\"%s\"", value)
				err := d.updateTXTRecord(recordName, quotedValue, ttl, "ACME challenge record")
				if err != nil {
					Warn("update failed, will retry",
						"uid", ctx.UID,
						"action", ctx.Action,
						"dns_name", ctx.DNSName,
						"record_name", recordName,
						"error", err,
						"attempt", attempt+1)
					continue
				}
				Info("updated TXT record successfully",
					"uid", ctx.UID,
					"action", ctx.Action,
					"dns_name", ctx.DNSName,
					"record_name", recordName,
					"value", value,
					"step", "complete")
				return nil
			}
			// Record has no value - delete and recreate
			Warn("existing record has no value, deleting and recreating",
				"uid", ctx.UID,
				"action", ctx.Action,
				"dns_name", ctx.DNSName,
				"record_id", *record.Id)
			_ = d.deleteRecordByID(*record.Id)
			continue

		default:
			// Multiple records exist - clean up all and recreate
			Warn("multiple records found, cleaning up and recreating",
				"uid", ctx.UID,
				"action", ctx.Action,
				"dns_name", ctx.DNSName,
				"record_name", recordName,
				"count", len(existingRecords),
				"step", "cleanup_duplicate",
			)

			for _, r := range existingRecords {
				if r.Id != nil {
					_ = d.deleteRecordByID(*r.Id)
				}
			}
			// Continue to next iteration to create new record
			continue
		}
	}

	return fmt.Errorf("failed to create TXT record after %d attempts", maxRetries)
}

// createTXTRecordWithRetry creates a new TXT record with conflict detection
func (d *DNSClient) createTXTRecordWithRetry(recordName, value string, ttl int) error {
	quotedValue := fmt.Sprintf("\"%s\"", value)
	description := "ACME challenge record"
	ttlValue := int32(ttl)

	request := &model.CreateRecordSetRequest{
		ZoneId: d.zoneID,
		Body: &model.CreateRecordSetRequestBody{
			Name:        recordName,
			Type:        "TXT",
			Ttl:         &ttlValue,
			Records:     []string{quotedValue},
			Description: &description,
		},
	}

	_, err := d.client.CreateRecordSet(request)
	return err
}

// updateTXTRecord updates an existing TXT record with a new value
func (d *DNSClient) updateTXTRecord(recordName, quotedValue string, ttl int, description string) error {
	// Get existing TXT records for this record name
	records, err := d.listTXTRecords(recordName)
	if err != nil {
		return fmt.Errorf("failed to list existing TXT records: %w", err)
	}

	if len(records) == 0 {
		return fmt.Errorf("no existing TXT record found to update")
	}

	// Update the first matching record (there should typically be only one)
	record := records[0]
	if record.Id == nil {
		return fmt.Errorf("existing record has no ID")
	}

	ttlValue := int32(ttl)

	updateRequest := &model.UpdateRecordSetRequest{
		ZoneId:     d.zoneID,
		RecordsetId: *record.Id,
		Body: &model.UpdateRecordSetReq{
			Ttl:         &ttlValue,
			Records:     &[]string{quotedValue},
			Description: &description,
		},
	}

	_, err = d.client.UpdateRecordSet(updateRequest)
	if err != nil {
		return fmt.Errorf("failed to update TXT record: %w", err)
	}

	return nil
}

// DeleteTXTRecord deletes a TXT record by matching the value
func (d *DNSClient) DeleteTXTRecord(ctx OperationContext, fqdn, value string) error {
	Info("deleting TXT record",
		"uid", ctx.UID,
		"action", ctx.Action,
		"dns_name", ctx.DNSName,
		"record_name", fqdn,
		"step", "initialize",
	)

	// Get all TXT records for this FQDN
	recordName := d.extractRecordName(fqdn)
	records, err := d.listTXTRecords(recordName)
	if err != nil {
		Error("failed to list TXT records",
			"uid", ctx.UID,
			"action", ctx.Action,
			"dns_name", ctx.DNSName,
			"record_name", recordName,
			"step", "list_records",
			"error", err)
		return fmt.Errorf("failed to list TXT records: %w", err)
	}

	// Find and delete the record matching the value
	for _, record := range records {
		if record.Records != nil && len(*record.Records) > 0 {
			// Remove quotes from stored value for comparison
			storedValue := strings.Trim((*record.Records)[0], `"`)
			if storedValue == value {
				Info("deleting TXT record by ID",
					"uid", ctx.UID,
					"action", ctx.Action,
					"dns_name", ctx.DNSName,
					"record_id", *record.Id,
					"record_name", recordName,
					"step", "delete",
				)
				err := d.deleteRecordByID(*record.Id)
				if err != nil {
					return err
				}
				Info("TXT record deleted",
					"uid", ctx.UID,
					"action", ctx.Action,
					"dns_name", ctx.DNSName,
					"record_id", *record.Id,
					"record_name", recordName,
					"step", "complete")
				return nil
			}
		}
	}

	// If we didn't find the exact record, return nil (idempotent)
	Debug("TXT record not found for deletion (idempotent)",
		"uid", ctx.UID,
		"action", ctx.Action,
		"dns_name", ctx.DNSName,
		"record_name", recordName)
	return nil
}

// listTXTRecords lists all TXT records for a given record name
func (d *DNSClient) listTXTRecords(recordName string) ([]model.ListRecordSets, error) {
	request := &model.ListRecordSetsByZoneRequest{
		ZoneId: d.zoneID,
	}

	response, err := d.client.ListRecordSetsByZone(request)
	if err != nil {
		return nil, fmt.Errorf("failed to list record sets: %w", err)
	}

	if response.Recordsets == nil {
		return []model.ListRecordSets{}, nil
	}

	// Filter for TXT records matching the record name
	// Note: API returns names with trailing dots, normalize for comparison
	var result []model.ListRecordSets
	for _, record := range *response.Recordsets {
		if record.Type != nil && *record.Type == "TXT" &&
			record.Name != nil && strings.TrimSuffix(*record.Name, ".") == recordName {
			result = append(result, record)
		}
	}

	return result, nil
}

// deleteRecordByID deletes a record by its ID
func (d *DNSClient) deleteRecordByID(recordID string) error {
	request := &model.DeleteRecordSetRequest{
		ZoneId:     d.zoneID,
		RecordsetId: recordID,
	}

	_, err := d.client.DeleteRecordSet(request)
	if err != nil {
		return fmt.Errorf("failed to delete record: %w", err)
	}

	return nil
}

// extractRecordName extracts the record name from FQDN by removing zone suffix
func (d *DNSClient) extractRecordName(fqdn string) string {
	// Ensure both have consistent trailing dots
	fqdn = strings.TrimSuffix(fqdn, ".")
	zoneName := strings.TrimSuffix(d.zoneName, ".")

	// Check if fqdn ends with zone name
	if strings.HasSuffix(fqdn, "."+zoneName) {
		return fqdn
	}

	// Return as-is if pattern doesn't match
	return fqdn
}
